package service

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"prometheus-deepflow-adapter/pkg/config"
	"prometheus-deepflow-adapter/pkg/log"
	"prometheus-deepflow-adapter/pkg/plugins/election"
)

type Service struct {
	conf    *config.Config
	engine  *gin.Engine
	elector election.Election

	lastReceiveTime           int64
	stopLockerLivenessChecker chan bool
	stopLivenessCheck         *atomic.Bool
}

func NewService(config *config.Config) *http.Server {
	gin.SetMode(config.Mode)
	s := &Service{
		engine:                    gin.Default(),
		conf:                      config,
		lastReceiveTime:           time.Now().UnixNano(),
		stopLockerLivenessChecker: make(chan bool),
		stopLivenessCheck:         &atomic.Bool{},
	}
	s.injectRouters()

	if config.ElectionEnabled {
		log.Logger.Info("msg", "election enabled, start server election")
		// TODO: can start tracing & inject context
		ctx := context.Background()
		elector := election.StartElection(config)
		if elector.IsLeader() {
			log.Logger.Debug("msg", "current server is leader now, start remote write")
		}
		go elector.KeepAlive(ctx)
		go s.lockerLivenessCheck(ctx)
	}

	if s.conf.PrometheusTimeout > 0 {
		ctx := context.Background()
		go s.prometheusLivenessCheck(ctx)
	}

	return &http.Server{Addr: config.Port, Handler: s.engine}
}

func (s *Service) injectRouters() {
	s.engine.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Invalid path: %s", c.Request.URL.Path)
	})
	s.engine.HandleMethodNotAllowed = true
	s.engine.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "Method not allowed: %s %s", c.Request.Method, c.Request.URL.Path)
	})

	router := s.engine.Group("")
	router.GET("/healthz", healthz())
	router.POST("/receive", prometheusLiveness(&s.lastReceiveTime,
		func() bool { return s.conf.ElectionEnabled && !s.elector.IsLeader() }),
		sendSamples(s.conf.RemoteWriteConfig.Url))
}

func (s *Service) Cleanup() error {
	// TODO: add resource cleanup
	log.Logger.Info("msg", "service cleanup start")
	return nil
}
