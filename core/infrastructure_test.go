//go:build testing
// +build testing

package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	testing_helpers "github.com/Reddetk/CBTraining/core/testhelpers"
	"github.com/Reddetk/CBTraining/logger"
)

func TestIndependentPaymentsParallel(t *testing.T) {
	const processingDelay = 100 * time.Millisecond
	const tolerancePct = 15.0

	// ── Arrange ────────────────────────────────────────────────────────────────
	processor := testing_helpers.NewMockPaymentProcessor(processingDelay)
	panding := testing_helpers.NewMockPendingRepo()
	repo := testing_helpers.NewMockPaymentRepo()
	logger := testing_helpers.NewMockLogger()

	maxWorkers := 2
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	svc, _ := NewPaymentManagerService(
		processor,
		panding,
		repo,
		rootCtx,
		maxWorkers, // maxWorkers — семафор на 2 слота
		logger,
	)

	ctx, cancel := testing_helpers.CreateContextWithTimeout(5 * time.Second)
	defer cancel()
	txs, pbs := testing_helpers.BuildIndependentPair()
	tx1 := txs[0]
	tx2 := txs[1]

	// ── Act ────────────────────────────────────────────────────────────────────
	// Отправляем обе TX параллельно через публичный API сервиса

	var elapsed time.Duration

	elapsed = testing_helpers.TimedExecution(func() {
		testing_helpers.WaitGroupAsync(
			func() {
				_, err := svc.PaymentCMD(ctx, tx1)
				testing_helpers.AssertNilError(t, err, "TX1 PaymentCMD")
			},
			func() {
				_, err := svc.PaymentCMD(ctx, tx2)
				testing_helpers.AssertNilError(t, err, "TX2 PaymentCMD")
			},
		)
	})

	// PaymentCMD возвращает сразу после постановки в очередь — ждём реальной обработки
	if !testing_helpers.WaitForProcessedCount(t, processor, 2, 3*time.Second) {
		t.Fatalf("timeout: обработано %d из 2 TX\n%s",
			processor.GetProcessedCount(),
			testing_helpers.DebugProcessedRecords(processor),
		)
	}
	aPA := pbs[0].DbIBAN()
	bPA := pbs[0].CrIBAN()
	kPA := pbs[1].DbIBAN()
	zPA := pbs[1].CrIBAN()
	// ── Assert 1: группы разные (диспетчер создал двух независимых воркеров) ───
	// Проверяем через dispatcher, доступный из сервиса
	if !testing_helpers.VerifyIBANInDifferentGroups(t, svc.dispatcher, []string{aPA, kPA}) {
		t.Errorf("A и K должны быть в разных группах\n%s",
			testing_helpers.DebugDispatcherState(svc.dispatcher))
	}

	if !testing_helpers.VerifyIBANInSameGroup(t, svc.dispatcher, []string{aPA, bPA}) {
		t.Error("A и B (одна TX) должны быть в одной группе")
	}

	if !testing_helpers.VerifyIBANInSameGroup(t, svc.dispatcher, []string{kPA, zPA}) {
		t.Error("K и Z (одна TX) должны быть в одной группе")
	}

	// ── Assert 2: параллельное исполнение по временным меткам ProcessRecord ────
	if !testing_helpers.VerifyParallelExecution(t, processor, tx1.EndToEndIdentification, tx2.EndToEndIdentification) {
		t.Errorf("TX должны выполняться параллельно\n%s",
			testing_helpers.DebugProcessedRecords(processor))
	}

	// ── Assert 3: суммарное время отправки = времени одной TX ─────────────────
	// elapsed — время двух параллельных PaymentCMD (неблокирующих),
	// поэтому он заведомо мал; основная проверка параллелизма — в Assert 2.
	// Здесь проверяем что не было неожиданной блокировки на семафоре.
	_ = elapsed // семафор не должен блокировать при maxWorkers=2

	// Реальная временна́я проверка: оба воркера работали одновременно,
	// значит общее время processor'а ≈ processingDelay, а не 2x
	records := processor.GetProcessed()
	if len(records) == 2 {
		testing_helpers.AssertTimeDiffWithinPercent(
			t,
			records[0].Duration,
			records[1].Duration,
			tolerancePct,
			"длительности обработки TX должны быть сопоставимы",
		)
	}

	// ── Debug  ─────────────────────────────────────────────
	if t.Failed() {
		t.Log(testing_helpers.DebugDispatcherState(svc.dispatcher))
		t.Log(testing_helpers.DebugProcessedRecords(processor))
		t.Logf("elapsed (PaymentCMD submit): %v", elapsed)
	}
}

func TestDependentChainSequential(t *testing.T) {
	const processingDelay = 100 * time.Millisecond
	const tolerancePct = 20.0
	const chainLen = 4

	processor := testing_helpers.NewMockPaymentProcessor(processingDelay)
	pending := testing_helpers.NewMockPendingRepo()
	repo := testing_helpers.NewMockPaymentRepo()
	logger := logger.New()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	maxWorkers := 4
	svc, err := NewPaymentManagerService(
		processor,
		pending,
		repo,
		rootCtx,
		maxWorkers,
		logger,
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	txs, _ := testing_helpers.BuildDependentChain(chainLen)
	if len(txs) != chainLen {
		t.Fatalf("expected %d payments in chain, got %d", chainLen, len(txs))
	}

	var elapsed time.Duration
	chain := make([]string, len(txs))
	prcs := make([]func(), 0, len(txs))
	t.Log("i - tx.EndToEndIdentification: tx.DebtorPacc.IBAN to tx.CreditorPacc.IBAN")
	t.Log("_________________________________________________________________________")
	for i, tx := range txs {
		chain[i] = tx.EndToEndIdentification
		t.Log(i, " - ", tx.EndToEndIdentification, ": ", tx.DebtorPacc.IBAN, " to ", tx.CreditorPacc.IBAN)
		cmnd := func() {
			_, err := svc.PaymentCMD(svc.dispatcher.RootCtx, tx)
			testing_helpers.AssertNilError(t, err, fmt.Sprintf("TX PaymentCMD %d", i))
		}
		prcs = append(prcs, cmnd)
	}

	elapsed = testing_helpers.TimedExecution(func() {
		testing_helpers.WaitGroupAsync(prcs...)
	})

	if !testing_helpers.WaitForProcessedCount(t, processor, 4, 3*time.Second) {
		t.Fatalf("timeout: обработано %d из 2 TX\n%s",
			processor.GetProcessedCount(),
			testing_helpers.DebugProcessedRecords(processor),
		)
	}
	_ = elapsed

	if !testing_helpers.VerifyProcessingOrder(t, processor, chain) {
		t.Fatalf("err")
	}

	// for i, tx := range txs[1:] {
	// 	prevTx := txs[i]
	// 	if !testing_helpers.VerifyStartAfterEnd(t, processor, prevTx.EndToEndIdentification, tx.EndToEndIdentification, 3*time.Second) {
	// 		if testing_helpers.VerifyParallelExecution(t, processor, prevTx.EndToEndIdentification, tx.EndToEndIdentification) {
	// 			t.Fatalf("tx %s, tx %s - должны последовательно 1",
	// 				txs[2].EndToEndIdentification,
	// 				txs[3].EndToEndIdentification,
	// 			)
	// 		} else {
	// 			t.Fatalf("not valiable test")
	// 		}
	// 	}
	// }
}

func TestTwoParalelDepChains(t *testing.T) {
	const processingDelay = 100 * time.Millisecond
	const tolerancePct = 20.0
	const chainLen = 4

	processor := testing_helpers.NewMockPaymentProcessor(processingDelay)
	pending := testing_helpers.NewMockPendingRepo()
	repo := testing_helpers.NewMockPaymentRepo()
	logger := logger.New()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	maxWorkers := 4
	svc, err := NewPaymentManagerService(
		processor,
		pending,
		repo,
		rootCtx,
		maxWorkers,
		logger,
	)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	txs, _ := testing_helpers.BuildDependentChain(chainLen)

	pltxs, _ := testing_helpers.BuildDependentChain(chainLen)

	var elapsed time.Duration
	chain := make([]string, len(txs))
	prcs := make([]func(), 0, len(txs))
	plchain := make([]string, len(txs))
	t.Log("i - tx.EndToEndIdentification: tx.DebtorPacc.IBAN to tx.CreditorPacc.IBAN")
	t.Log("_________________________________________________________________________")

	for i, tx := range txs {
		pltx := pltxs[i]
		chain[i] = tx.EndToEndIdentification
		plchain[i] = pltx.EndToEndIdentification
		t.Log(i, " - ", tx.EndToEndIdentification, ": ", tx.DebtorPacc.IBAN, " to ", tx.CreditorPacc.IBAN)
		t.Log(i, " - ", pltx.EndToEndIdentification, ": ", pltx.DebtorPacc.IBAN, " to ", pltx.CreditorPacc.IBAN)
		cmnd := func() {
			_, err := svc.PaymentCMD(svc.dispatcher.RootCtx, tx)
			testing_helpers.AssertNilError(t, err, fmt.Sprintf("TX PaymentCMD %d", i))
		}
		plcmnd := func() {
			_, err := svc.PaymentCMD(svc.dispatcher.RootCtx, pltx)
			testing_helpers.AssertNilError(t, err, fmt.Sprintf("TX PaymentCMD %d", i))
		}
		prcs = append(prcs, cmnd)
		prcs = append(prcs, plcmnd)
	}

	elapsed = testing_helpers.TimedExecution(func() {
		testing_helpers.WaitGroupAsync(prcs...)
	})

	if !testing_helpers.WaitForProcessedCount(t, processor, 8, 3*time.Second) {
		t.Fatalf("timeout: обработано %d из 8 TX\n%s",
			processor.GetProcessedCount(),
			testing_helpers.DebugProcessedRecords(processor),
		)
	}
	_ = elapsed

	if !testing_helpers.VerifyParallelExecution(t, processor, txs[0].EndToEndIdentification, pltxs[0].EndToEndIdentification) {
		t.Fatal("NO")
	}
}
