package http

import (
	_ "github.com/Reddetk/CBTraining/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(r *gin.Engine, h *Handler) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/payments", h.CreatePayment)
		v1.GET("/payments/:txid", h.GetPaymentInfo)
	}
}
