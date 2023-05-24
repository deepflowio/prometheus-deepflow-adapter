package service

import (
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"prometheus-deepflow-adapter/pkg/log"
)

func healthz() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	}
}

func prometheusLiveness(lastReceiveTime *int64, abortExecute func() bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		atomic.StoreInt64(lastReceiveTime, time.Now().UnixNano())
		if abortExecute() {
			log.Logger.Info("msg", "server is not leader, abort remote write")
			c.AbortWithStatus(204)
		} else {
			log.Logger.Debug("msg", "promtheus remote write execution")
		}
	}
}
