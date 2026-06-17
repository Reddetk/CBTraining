//go:build testing
// +build testing

package core

import (
	"context"
	"testing"
	"time"

	testing_helpers "github.com/Reddetk/CBTraining/core/testhelpers"
	"github.com/Reddetk/CBTraining/logger"
	inport "github.com/Reddetk/CBTraining/ports/inports"
)

const (
	processingDelay = 100 * time.Millisecond
	testTimeout     = 10 * time.Second
	maxWorkers      = 5
)

func newTestService(t *testing.T, delay time.Duration) (
	*PaymentManagerService,
	*testing_helpers.MockPaymentProcessor,
	logger.Logger,
) {
	t.Helper()
	processor := testing_helpers.NewMockPaymentProcessor(delay)
	pending := testing_helpers.NewMockPendingRepo()
	repo := testing_helpers.NewMockPaymentRepo()
	log := logger.New()

	rootCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	svc, err := NewPaymentManagerService(processor, pending, repo, rootCtx, maxWorkers, log)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	return svc, processor, log
}

func submitAll(t *testing.T, svc *PaymentManagerService, txs []*inport.PaymentRequest) []string {
	t.Helper()
	etes := make([]string, len(txs))
	cmds := make([]func(), len(txs))
	for i, tx := range txs {
		i, tx := i, tx
		etes[i] = tx.EndToEndIdentification
		cmds[i] = func() {
			_, err := svc.PaymentCMD(svc.dispatcher.RootCtx, tx)
			if err != nil {
				t.Logf("PaymentCMD error: %v", err) // посмотри что именно
			}
		}
	}
	testing_helpers.WaitGroupAsync(cmds...)
	return etes
}

// ─────────────────────────────────────────────────────────────────────────────
// ИНВАРИАНТ 1: SAFETY
// ─────────────────────────────────────────────────────────────────────────────

// TestSafety_DependentChain — цепочка A→B, B→C, C→D: все конфликтуют,
// ни один не должен выполняться параллельно с соседом
func TestSafety_DependentChain(t *testing.T) {
	const chainLen = 4
	svc, processor, log := newTestService(t, processingDelay)

	txs, _ := testing_helpers.BuildDependentChain(chainLen)
	etes := submitAll(t, svc, txs)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("timeout\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	records := processor.GetProcessed()
	g := testing_helpers.BuildConflictGraph(records)
	testing_helpers.AssertSafety(t, g)
}

// TestSafety_ConcurrentPressure — N платежей конкурируют за один общий ЦС:
// в каждый момент обрабатывается ровно один
func TestSafety_ConcurrentPressure(t *testing.T) {
	const count = 6
	svc, processor, log := newTestService(t, processingDelay)

	txs, _ := testing_helpers.BuildConcurrentPressure(count)
	etes := submitAll(t, svc, txs)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("timeout\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	records := processor.GetProcessed()
	g := testing_helpers.BuildConflictGraph(records)
	testing_helpers.AssertSafety(t, g)
}

// ─────────────────────────────────────────────────────────────────────────────
// ИНВАРИАНТ 2: LIVENESS
// ─────────────────────────────────────────────────────────────────────────────

// TestLiveness_AllProcessed — все поданные платежи должны завершиться
func TestLiveness_AllProcessed(t *testing.T) {
	svc, processor, log := newTestService(t, processingDelay)

	// Смешиваем зависимые и независимые
	depTxs, _ := testing_helpers.BuildDependentChain(3)
	indTxs, _ := testing_helpers.BuildIndependentPair()
	all := testing_helpers.MergePaymentRequests(depTxs, indTxs)

	etes := submitAll(t, svc, all)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("starvation detected\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	testing_helpers.AssertLiveness(t, processor.GetProcessed(), etes)
}

// TestLiveness_HighConcurrency — при высокой нагрузке ни один платёж не теряется
func TestLiveness_HighConcurrency(t *testing.T) {
	svc, processor, log := newTestService(t, 50*time.Millisecond)

	pressure, _ := testing_helpers.BuildConcurrentPressure(10)
	ind1, _ := testing_helpers.BuildIndependentPair()
	ind2, _ := testing_helpers.BuildIndependentPair()
	all := testing_helpers.MergePaymentRequests(pressure, ind1, ind2)

	etes := submitAll(t, svc, all)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("timeout at high concurrency\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	testing_helpers.AssertLiveness(t, processor.GetProcessed(), etes)
}

// ─────────────────────────────────────────────────────────────────────────────
// ИНВАРИАНТ 3: PARALLELISM
// ─────────────────────────────────────────────────────────────────────────────

// TestParallelism_IndependentPair — два платежа без общих ЦС должны идти параллельно
func TestParallelism_IndependentPair(t *testing.T) {
	// Увеличиваем delay чтобы гарантировать видимое перекрытие
	svc, processor, log := newTestService(t, 300*time.Millisecond)

	txs, _ := testing_helpers.BuildIndependentPair()
	etes := submitAll(t, svc, txs)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("timeout\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	records := processor.GetProcessed()
	g := testing_helpers.BuildConflictGraph(records)
	testing_helpers.AssertParallelism(t, g)
}

// TestParallelism_MixedLoad — зависимые идут последовательно, независимые параллельно
func TestParallelism_MixedLoad(t *testing.T) {
	svc, processor, log := newTestService(t, 300*time.Millisecond)

	// A→B, B→C — конфликтуют между собой
	depTxs, _ := testing_helpers.BuildDependentChain(2)
	// K→Z — не пересекается ни с кем
	indTxs, _ := testing_helpers.BuildIndependentPair()

	all := testing_helpers.MergePaymentRequests(depTxs, indTxs)
	etes := submitAll(t, svc, all)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("timeout\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	records := processor.GetProcessed()
	g := testing_helpers.BuildConflictGraph(records)

	// Safety: конфликтующие не пересекаются
	testing_helpers.AssertSafety(t, g)
	// Parallelism: независимые шли параллельно
	testing_helpers.AssertParallelism(t, g)
}

// ─────────────────────────────────────────────────────────────────────────────
// КОМБО: ВСЕ ИНВАРИАНТЫ
// ─────────────────────────────────────────────────────────────────────────────

// TestAllInvariants_FullScenario — сценарий из ТЗ: A→B, B→C, K→Z
func TestAllInvariants_FullScenario(t *testing.T) {
	svc, processor, log := newTestService(t, 300*time.Millisecond)

	sharedIBAN := "BY99TEST000000000000000001"
	txs, _ := testing_helpers.BuildMany([]testing_helpers.IBANPair{
		{Debtor: "BY99TEST000000000000000000", Creditor: sharedIBAN},                   // A→B
		{Debtor: sharedIBAN, Creditor: "BY99TEST000000000000000002"},                   // B→C (ждёт)
		{Debtor: "BY99TEST000000000000000003", Creditor: "BY99TEST000000000000000004"}, // K→Z (независим)
	})

	etes := submitAll(t, svc, txs)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("timeout\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	testing_helpers.AssertAllInvariants(t, processor, etes)
}

// TestAllInvariants_DependentChain — регрессия: цепочка из 4, все инварианты
func TestAllInvariants_DependentChain(t *testing.T) {
	const chainLen = 4
	svc, processor, log := newTestService(t, processingDelay)

	txs, _ := testing_helpers.BuildDependentChain(chainLen)
	etes := submitAll(t, svc, txs)

	if !testing_helpers.WaitLiveness(t, processor, etes, testTimeout) {
		t.Fatalf("timeout\n%s", testing_helpers.DebugProcessedRecords(processor))
	}

	processor.VisualizeTime(log)
	testing_helpers.AssertAllInvariants(t, processor, etes)

	// Дополнительно: порядок в цепочке соблюдён
	testing_helpers.VerifyProcessingOrder(t, processor, etes)
}
