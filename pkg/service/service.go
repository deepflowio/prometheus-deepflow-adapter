package service

import (
	"context"
	"fmt"
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

	lastReceiveTime   int64
	stopLivenessCheck *atomic.Bool
	retryLock         *time.Ticker
	keepAlive         *time.Ticker
}

func NewService(config *config.Config) *http.Server {
	s := &Service{
		engine:            gin.Default(),
		conf:              config,
		lastReceiveTime:   time.Now().UnixNano(),
		stopLivenessCheck: &atomic.Bool{},
	}
	s.injectMiddlewares()
	s.injectRouters()

	if config.ElectionEnabled {
		log.Logger.Info("msg", "election enabled, start server election")
		// TODO: start tracing & inject span to context
		ctx := context.Background()
		s.elector = election.StartElection(config)
		s.keepAlive = time.NewTicker(s.elector.HeartBeat())
		s.retryLock = time.NewTicker(s.elector.RetryPeriod())
		if s.elector.IsLeader() {
			log.Logger.Debug("msg", "current server is leader now, start remote write")
			go s.lockerKeepAlive(ctx)
		} else {
			log.Logger.Debug("msg", "current server is not leader, start retry for leader release")
			go s.lockerRetry(ctx)
		}
	}

	if s.conf.PrometheusScrapeInterval > 0 {
		ctx := context.Background()
		go s.prometheusLivenessCheck(ctx)
	}
	svc := &http.Server{Addr: fmt.Sprintf(":%d", config.Port), Handler: s.engine}
	svc.RegisterOnShutdown(func() {
		ctx := context.Background()
		err := s.Cleanup(ctx)
		if err != nil {
			log.Logger.Error("msg", "cleanup failed", "err", err)
		}
	})
	return svc
}

func (s *Service) injectMiddlewares() {
	s.engine.Use(gin.LoggerWithWriter(log.Logger))
	s.engine.Use(gin.LoggerWithFormatter(func(params gin.LogFormatterParams) string {
		return fmt.Sprintf("%s -\"%s %s %s %d %s \"%s\" %s\"",
			params.ClientIP,
			params.Method,
			params.Path,
			params.Request.Proto,
			params.StatusCode,
			params.Latency,
			params.Request.UserAgent(),
			params.ErrorMessage)
	}))
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

func (s *Service) Cleanup(ctx context.Context) error {
	log.Logger.Info("msg", "service cleanup start")
	err := s.elector.Release(ctx)
	if err != nil {
		return err
	}
	log.Logger.Info("msg", "service cleanup complete")
	return nil
}
