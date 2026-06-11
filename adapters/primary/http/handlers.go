// Package http implement primary adapter Gin router
package http

import (
	"net/http"

	inport "github.com/Reddetk/CBTraining/ports/inports"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	payManager inport.PayManager
	payAuditor inport.PayAuditor
}

func NewHandler(payManager inport.PayManager, payAuditor inport.PayAuditor) *Handler {
	return &Handler{
		payManager: payManager,
		payAuditor: payAuditor,
	}
}

// CreatePayment godoc
// @Summary      Send payment to processing
// @Description  Initiates a payment transaction and sends it to the processing system
// @Tags         payments
// @Accept       json
// @Produce      json
// @Param        request body inport.PaymentRequest true "Payment request payload"
// @Success      200  {object}  inport.TXConfirmation
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/payments [post]
func (h *Handler) CreatePayment(c *gin.Context) {
	var req inport.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	confirmation, err := h.payManager.PaymentCMD(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, confirmation)
}

// GetPaymentInfo godoc
// @Summary      Get payment status
// @Description  Returns current status and details of a payment by TXID
// @Tags         payments
// @Produce      json
// @Param        txid path string true "Transaction ID"
// @Success      200  {object}  inport.PaymentInfo
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/payments/{txid} [get]
func (h *Handler) GetPaymentInfo(c *gin.Context) {
	txid := c.Param("txid")

	info, err := h.payAuditor.GetPaymentInfo(c.Request.Context(), txid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	if info == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "payment not found"})
		return
	}

	c.JSON(http.StatusOK, info)
}

type ErrorResponse struct {
	Error string `json:"error"`
}
