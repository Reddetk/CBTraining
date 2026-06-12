// Package core contains the core business logic and domain entities for the payment processing system. It defines the main services, such as PaymentManagerService and PaymentAuditService, which handle the core functionalities of managing payments and auditing transactions. The core package also includes the essential data structures and error definitions used throughout the application.
package core

import (
	"context"

	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
	"github.com/Reddetk/CBTraining/logger"
	inport "github.com/Reddetk/CBTraining/ports/inports"
	outport "github.com/Reddetk/CBTraining/ports/outports"
	"go.uber.org/zap"
)

type PaymentAuditService struct {
	repo   outport.PaymentRepo
	logger logger.Logger
}

func NewPaymentAuditService(repo outport.PaymentRepo, logger logger.Logger) *PaymentAuditService {
	return &PaymentAuditService{repo: repo, logger: logger}
}

func (as *PaymentAuditService) GetPaymentInfo(ctx context.Context, TXID string) (*inport.PaymentInfo, error) {
	TXRec, err := as.repo.GetTXByID(ctx, TXID)
	if err != nil {
		return nil, corerr.ErrInfrastructure
	}
	if TXRec == nil {
		as.logger.Info("payment not found", zap.String("txid", TXID))
		return nil, corerr.ErrPaymentNotFound
	}
	return &inport.PaymentInfo{
		TXID:            TXRec.TXID,
		Status:          TXRec.Status,
		CreditorAccount: &inport.PaymentAccountDTO{IBAN: TXRec.CreditorPacc.IBAN, AccCurency: TXRec.CreditorPacc.AccCurency},
		DebitorAccount:  &inport.PaymentAccountDTO{IBAN: TXRec.DebitorPacc.IBAN, AccCurency: TXRec.DebitorPacc.AccCurency},
		Metadata:        TXRec.Metadata,
	}, nil
}
