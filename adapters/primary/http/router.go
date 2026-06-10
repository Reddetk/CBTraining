// Package http implement primary adapter Gin router
package http

import "github.com/gin-gonic/gin"

type GinRouter struct {
	router *gin.Engine
}

func NewGinRouter() *GinRouter {
	return &GinRouter{
		router: gin.Default(),
	}
}
