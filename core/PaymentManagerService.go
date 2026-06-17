package core

import (
	"context"
	"fmt"

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
		return fmt.Errorf("error of recover panding %w", corerr.ErrInfrastructure)
	}
	for _, TXrec := range TXrecs {
		tx, err := entity.RecoverPaymentTXFromRec(TXrec)
		if err != nil {
			ps.log.Error("error of recover pay TX from rec", zap.Error(err))
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
		return nil, corerr.ErrInfrastructure
	}

	if err := ps.add(tx); err != nil {
		ps.log.Error(err.Error())
		return nil, err
	}

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
			ps.log.Info("add to group", zap.String("ETE", tx.EndToEndIdentification.Identification))
			g = gA
			if g == nil {
				g = gB
			}
			ps.dispatcher.IbanToGroup[tx.DebIBAN()] = g
			ps.dispatcher.IbanToGroup[tx.CredIBAN()] = g

			<-ps.dispatcher.Semaphore
		} else {
			ps.log.Info("New Group", zap.String("ETE", tx.EndToEndIdentification.Identification))
			g = entity.NewGroup(ps.dispatcher.RootCtx)
			ps.dispatcher.IbanToGroup[tx.DebIBAN()] = g
			ps.dispatcher.IbanToGroup[tx.CredIBAN()] = g
			ps.dispatcher.Wg.Add(1)
			go ps.runWorker(g)
		}
		ps.dispatcher.Mu.Unlock()

	case gA != nil && gB == nil:
		g = gA
		ps.dispatcher.IbanToGroup[tx.DebIBAN()] = g
		ps.log.Info("connect to group", zap.String("ETE", tx.EndToEndIdentification.Identification))
		ps.dispatcher.Mu.Unlock()

	case gA == nil && gB != nil:
		g = gB
		ps.dispatcher.IbanToGroup[tx.CredIBAN()] = g
		ps.log.Info("connect to group", zap.String("ETE", tx.EndToEndIdentification.Identification))
		ps.dispatcher.Mu.Unlock()

	case gA == gB:
		ps.log.Info("connect to group", zap.String("ETE", tx.EndToEndIdentification.Identification))
		g = gA
		ps.dispatcher.Mu.Unlock()

	default:

		ps.log.Info("MERGE ", zap.String("ETE", tx.EndToEndIdentification.Identification))
		g = gA
		gB.Merging.Store(true)

		for iban, grp := range ps.dispatcher.IbanToGroup {
			if grp == gB {
				ps.dispatcher.IbanToGroup[iban] = gA
			}
		}

		gB.WorkerCancelSig()
		ps.dispatcher.Mu.Unlock()

		select {
		case <-gB.DrainReady:
		case <-ps.dispatcher.RootCtx.Done():
			return corerr.ErrDispatcherShuting
		}

		ps.dispatcher.Mu.Lock()
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

	select {
	case g.Ch <- tx:
		return nil
	case <-ps.dispatcher.RootCtx.Done():
		return corerr.ErrDispatcherShuting
	}
}

func (ps *PaymentManagerService) runWorker(g *entity.Group) {
	defer ps.dispatcher.Wg.Done()
	defer func() { <-ps.dispatcher.Semaphore }()

	for {
		select {
		case tx := <-g.Ch:
			ps.log.Info("tx sended to procesTX: ", zap.String("ETE", tx.EndToEndIdentification.Identification))
			err := ps.paymentProc.ProcessTX(ps.dispatcher.RootCtx, tx.ToTxRequest())
			if err != nil {
				ps.log.Error(err.Error())
			}
		case <-g.GrCtx.Done():
			if g.Merging.Load() {
				close(g.DrainReady)
				return
			}
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
	if err := ps.paymentRepo.PersistProcessResult(ctx, payRes.TXID, payRes.Result); err != nil {
		return fmt.Errorf("error of persist processing result %w", corerr.ErrInfrastructure)
	}
	return nil
}

func (ps *PaymentManagerService) Shutdown() {
	ps.dispatcher.Wg.Wait()
}
