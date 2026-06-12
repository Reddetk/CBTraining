package core

import (
	"context"

	"github.com/Reddetk/CBTraining/core/entity"
	"github.com/Reddetk/CBTraining/logger"
	outport "github.com/Reddetk/CBTraining/ports/outports"
)

type PaymentManagerService struct {
	pandingRepo outport.PendingRepo
	paymentRepo outport.PaymentRepo
	dispatcher  *entity.Dispatcher
	log         logger.Logger
}

func NewPaymentManagerService(
	payProc outport.PaymentProcessor,
	panRep outport.PendingRepo,
	payRep outport.PaymentRepo,
	rootCtx context.Context,
	maxWorker int,
	log logger.Logger,
) (*PaymentManagerService, error) {
	processFn := func(tx *entity.PaymentTX) error {
		// TODO implement
		return nil
	}

	disp := entity.NewDispatcher(rootCtx, maxWorker, processFn)

	return &PaymentManagerService{
		pandingRepo: panRep,
		paymentRepo: payRep,
		dispatcher:  disp,
		log:         log,
	}, nil
}

// func (d *Dispatcher) Add(tx *TX) error {
// 	d.mu.Lock()

// 	// Проверяем не завершён ли диспетчер
// 	select {
// 	case <-d.ctx.Done():
// 		d.mu.Unlock()
// 		return fmt.Errorf("dispatcher is shutting down")
// 	default:
// 	}

// 	gA := d.ibanToGroup[tx.IBANfrom]
// 	gB := d.ibanToGroup[tx.IBANto]

// 	var g *Group

// 	switch {
// 	case gA == nil && gB == nil:
// 		// Захватываем слот семафора не блокируя мьютекс
// 		d.mu.Unlock()
// 		select {
// 		case d.semaphore <- struct{}{}:
// 		case <-d.ctx.Done():
// 			return fmt.Errorf("dispatcher is shutting down")
// 		}
// 		d.mu.Lock()
// 		// Перепроверяем после повторного захвата мьютекса —
// 		// пока ждали семафор другая горутина могла создать группу
// 		gA = d.ibanToGroup[tx.IBANfrom]
// 		gB = d.ibanToGroup[tx.IBANto]
// 		if gA != nil || gB != nil {
// 			// Группа появилась пока ждали — рекурсивно не идём,
// 			// просто добавляем в существующую
// 			g = gA
// 			if g == nil {
// 				g = gB
// 			}
// 			d.ibanToGroup[tx.IBANfrom] = g
// 			d.ibanToGroup[tx.IBANto] = g
// 			// Возвращаем слот — новый воркер не нужен
// 			<-d.semaphore
// 		} else {
// 			g = newGroup(d.ctx)
// 			d.ibanToGroup[tx.IBANfrom] = g
// 			d.ibanToGroup[tx.IBANto] = g
// 			d.wg.Add(1)
// 			go d.runWorker(g)
// 		}
// 		d.mu.Unlock()

// 	case gA != nil && gB == nil:
// 		g = gA
// 		d.ibanToGroup[tx.IBANto] = g
// 		d.mu.Unlock()

// 	case gA == nil && gB != nil:
// 		g = gB
// 		d.ibanToGroup[tx.IBANfrom] = g
// 		d.mu.Unlock()

// 	case gA == gB:
// 		g = gA
// 		d.mu.Unlock()

// 	default: // merge gA + gB
// 		g = gA
// 		// Останавливаем воркер gB через workerCancelSig — он завершится
// 		// после текущего TX, семафор освободит сам
// 		gB.workerCancelSig()
// 		// Дренируем буфер gB → gA пока держим мьютекс
// 	drain:
// 		for {
// 			select {
// 			case pending := <-gB.ch:
// 				gA.ch <- pending
// 			default:
// 				break drain
// 			}
// 		}
// 		for iban, grp := range d.ibanToGroup {
// 			if grp == gB {
// 				d.ibanToGroup[iban] = gA
// 			}
// 		}
// 		d.mu.Unlock()
// 	}

// 	// Неблокирующая отправка с учётом shutdown
// 	select {
// 	case g.ch <- tx:
// 		return nil
// 	case <-d.ctx.Done():
// 		return fmt.Errorf("dispatcher is shutting down")
// 	}
// }

// func (d *Dispatcher) runWorker(g *Group) {
// 	defer d.wg.Done()
// 	defer func() { <-d.semaphore }() // освобождаем слот при любом выходе

// 	for {
// 		select {
// 		case tx := <-g.ch:
// 			d.process(tx)

// 		case <-g.ctx.Done():
// 			// Группа слита или диспетчер завершается —
// 			// дочитываем оставшееся в буфере перед выходом
// 			for {
// 				select {
// 				case tx := <-g.ch:
// 					d.process(tx)
// 				default:
// 					return // буфер пуст — выходим
// 				}
// 			}
// 		}
// 	}
// }

// // Shutdown — graceful: ждём завершения всех воркеров
// func (d *Dispatcher) Shutdown() {
// 	// ctx диспетчера отменяется снаружи через workerCancelSig переданный в NewDispatcher
// 	d.wg.Wait()
// }

// // ──────────────────────────────────────────────
// // Main
// // ──────────────────────────────────────────────

// func main() {
// 	ctx, workerCancelSig := context.WithCancel(context.Background())

// 	var txWg sync.WaitGroup

// 	d := NewDispatcher(ctx, 5, func(tx *TX) {
// 		// fmt.Printf("  → start: %s (%s→%s)\n", tx.ID, tx.IBANfrom, tx.IBANto)
// 		time.Sleep(200 * time.Millisecond)
// 		fmt.Printf("  ✓ done:  %s\n", tx.ID)
// 		txWg.Done()
// 	})

// 	txs := []*TX{
// 		{ID: "tx-1", IBANfrom: "A", IBANto: "B"},
// 		{ID: "tx-2", IBANfrom: "B", IBANto: "C"},
// 		{ID: "tx-3", IBANfrom: "K", IBANto: "Z"},
// 		{ID: "tx-5", IBANfrom: "K", IBANto: "X"},
// 		{ID: "tx-6", IBANfrom: "X", IBANto: "Z"},
// 		{ID: "tx-10", IBANfrom: "F", IBANto: "L"},
// 	}

// 	txWg.Add(len(txs))

// 	for _, tx := range txs {
// 		if err := d.Add(tx); err != nil {
// 			fmt.Println("add error:", err)
// 		}
// 	}

// 	txWg.Wait()       // все TX обработаны
// 	workerCancelSig() // сигнал всем воркерам завершаться
// 	d.Shutdown()      // ждём чистого выхода горутин

// 	fmt.Println("\nвсе TX обработаны, диспетчер остановлен")
// }
