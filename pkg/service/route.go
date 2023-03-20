package service

import (
	"github.com/gin-gonic/gin"
	"prometheus-deepflow-adapter/pkg/config"
)

func (s *engine) injectRouterGroup(router *gin.RouterGroup) {

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	router.POST("/receive", ReceiveHandler(config.RemoteUrl))
}
