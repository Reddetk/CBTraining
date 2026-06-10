// Package core contains the core business logic and domain entities for the payment processing system. It defines the main services, such as PaymentManagerService and PaymentAuditService, which handle the core functionalities of managing payments and auditing transactions. The core package also includes the essential data structures and error definitions used throughout the application.
package core

import (
	"context"

	inport "github.com/Reddetk/CBTraining/ports/inports"
	outport "github.com/Reddetk/CBTraining/ports/outports"
)

type PaymentAuditService struct {
	repo outport.PaymentRepo
}

func NewPaymentAuditService(repo outport.PaymentRepo) *PaymentAuditService {
	return &PaymentAuditService{repo: repo}
}

func (as *PaymentAuditService) GetPaymentInfo(ctx context.Context, TXID string) (*inport.PaymentInfo, error) {
	return &inport.PaymentInfo{}, nil
}
