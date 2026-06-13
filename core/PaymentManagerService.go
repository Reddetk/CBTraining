package core

import (
	"context"

	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
	"github.com/Reddetk/CBTraining/core/entity"
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	"github.com/Reddetk/CBTraining/logger"
	inport "github.com/Reddetk/CBTraining/ports/inports"
	outport "github.com/Reddetk/CBTraining/ports/outports"
	"go.uber.org/zap"
)

type PaymentManagerService struct {
	paymentProc outport.PaymentProcessor
	pandRepo    outport.PendingRepo
	paymentRepo outport.PaymentRepo
	dispatcher  *entity.Dispatcher
	log         logger.Logger
}

func NewPaymentManagerService(
	payProc outport.PaymentProcessor,
	pandRepo outport.PendingRepo,
	payRep outport.PaymentRepo,
	rootCtx context.Context,
	maxWorker int,
	log logger.Logger,
) (*PaymentManagerService, error) {
	disp := entity.NewDispatcher(rootCtx, maxWorker)
	PayManServ := &PaymentManagerService{
		paymentProc: payProc,
		pandRepo:    pandRepo,
		paymentRepo: payRep,
		dispatcher:  disp,
		log:         log,
	}
	err := PayManServ.pandingRecovery(context.Background())
	if err != nil {
		return nil, err
	}
	PayManServ.log.Info("new pay manager server started")
	return PayManServ, nil
}

func (ps *PaymentManagerService) pandingRecovery(ctx context.Context) error {
	TXrecs, err := ps.pandRepo.LoadPendingPayments(ctx)
	if err != nil {
		ps.log.Error(err.Error())
		return err
	}
	for _, TXrec := range TXrecs {
		tx, err := entity.RecoverPaymentTXFromRec(TXrec)
		if err != nil {
			ps.log.Error(err.Error())
			continue
		}
		if err := ps.add(tx); err != nil {
			ps.log.Error(err.Error())
			continue
		}
	}
	ps.log.Info("panding recovery done")
	return nil
}

func (ps *PaymentManagerService) PaymentCMD(
	ctx context.Context,
	req *inport.PaymentRequest,
) (*inport.TXConfirmation, error) {
	tx, err := entity.NewPaymentTXFromDTO(*req)
	if err != nil {
		ps.log.Error(err.Error())
		return nil, err
	}

	err = ps.paymentRepo.InsertTX(ctx, tx.ToRecord())
	if err != nil {
		ps.log.Error(err.Error())
		return nil, err
	}

	if err := ps.add(tx); err != nil {
		ps.log.Error(err.Error())
		return nil, err
	}

	ps.log.Info("payment req sended", zap.String("txid", tx.TXID))
	return &inport.TXConfirmation{
		TXID:     tx.TXID,
		Status:   string(valobj.Pending),
		Metadata: tx.Metadata.Touch().String(),
	}, nil
}

func (ps *PaymentManagerService) add(tx *entity.PaymentTX) error {
	ps.dispatcher.Mu.Lock()

	select {
	case <-ps.dispatcher.RootCtx.Done():
		ps.dispatcher.Mu.Unlock()
		ps.log.Error(corerr.ErrDispatcherShuting.Error())
		return corerr.ErrDispatcherShuting
	default:
	}

	gA := ps.dispatcher.IbanToGroup[tx.CredIBAN()]
	gB := ps.dispatcher.IbanToGroup[tx.DebIBAN()]

	var g *entity.Group

	switch {
	case gA == nil && gB == nil:
		ps.dispatcher.Mu.Unlock()
		select {
		case ps.dispatcher.Semaphore <- struct{}{}:
		case <-ps.dispatcher.RootCtx.Done():
			return corerr.ErrDispatcherShuting
		}
		ps.dispatcher.Mu.Lock()

		gA = ps.dispatcher.IbanToGroup[tx.CredIBAN()]
		gB = ps.dispatcher.IbanToGroup[tx.DebIBAN()]
		if gA != nil || gB != nil {

			g = gA
			if g == nil {
				g = gB
			}
			ps.dispatcher.IbanToGroup[tx.DebIBAN()] = g
			ps.dispatcher.IbanToGroup[tx.CredIBAN()] = g

			<-ps.dispatcher.Semaphore
		} else {
			g = entity.NewGroup(ps.dispatcher.RootCtx)
			ps.dispatcher.IbanToGroup[tx.DebIBAN()] = g
			ps.dispatcher.IbanToGroup[tx.CredIBAN()] = g
			ps.dispatcher.Wg.Add(1)
			go ps.runWorker(g)
		}
		ps.dispatcher.Mu.Unlock()

	case gA != nil && gB == nil:
		g = gA
		ps.dispatcher.IbanToGroup[tx.CredIBAN()] = g
		ps.dispatcher.Mu.Unlock()

	case gA == nil && gB != nil:
		g = gB
		ps.dispatcher.IbanToGroup[tx.DebIBAN()] = g
		ps.dispatcher.Mu.Unlock()

	case gA == gB:
		g = gA
		ps.dispatcher.Mu.Unlock()

	default:
		g = gA
		gB.WorkerCancelSig()
	drain:
		for {
			select {
			case pending := <-gB.Ch:
				gA.Ch <- pending
			default:
				break drain
			}
		}
		for iban, grp := range ps.dispatcher.IbanToGroup {
			if grp == gB {
				ps.dispatcher.IbanToGroup[iban] = gA
			}
		}
		ps.dispatcher.Mu.Unlock()
	}

	// Неблокирующая отправка с учётом shutdown
	select {
	case g.Ch <- tx:
		return nil
	case <-ps.dispatcher.RootCtx.Done():
		return corerr.ErrDispatcherShuting
	}
}

func (ps *PaymentManagerService) runWorker(g *entity.Group) {
	defer ps.dispatcher.Wg.Done()
	defer func() { <-ps.dispatcher.Semaphore }() // освобождаем слот при любом выходе

	for {
		select {
		case tx := <-g.Ch:
			err := ps.paymentProc.ProcessTX(ps.dispatcher.RootCtx, tx.ToTxRequest()) // Передаем на вторичный адаптер
			if err != nil {
				ps.log.Error(err.Error())
			}
		case <-g.GrCtx.Done():
			for {
				select {
				case tx := <-g.Ch:
					err := ps.paymentProc.ProcessTX(ps.dispatcher.RootCtx, tx.ToTxRequest())
					if err != nil {
						ps.log.Error(err.Error())
					}
				default:
					return
				}
			}
		}
	}
}

func (ps *PaymentManagerService) StorePaymentResult(ctx context.Context, payRes *inport.PaymentResult) error {
	return ps.paymentRepo.PersistProcessResult(ctx, payRes.TXID, payRes.Result)
}

func (ps *PaymentManagerService) Shutdown() {
	ps.dispatcher.Wg.Wait()
}
