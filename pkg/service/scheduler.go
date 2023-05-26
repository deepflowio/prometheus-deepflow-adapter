package service

import (
	"context"
	"prometheus-deepflow-adapter/pkg/log"
	"time"
)

// temporary add magic timeout for prometheus remote_write
// TODO: make it configurable
const magicTimeout = 10 * time.Second

func (svc *Service) prometheusLivenessCheck(ctx context.Context) {
	ticker := time.NewTicker(magicTimeout)
	for range ticker.C {
		elapsed := time.Since(time.Unix(0, svc.lastReceiveTime))
		if elapsed > svc.conf.PrometheusScrapeInterval+magicTimeout {
			svc.keepAlive.Stop()
			svc.retryLock.Stop()
			svc.stopLivenessCheck.Store(true)

			if svc.elector.IsLeader() {
				// prometheus liveness check failed
				err := svc.elector.Release(ctx)
				if err != nil {
					log.Logger.Error("msg", "release elector locker failed", "err", err)
				}
				log.Logger.Info("msg", "prometheus liveness failed, release lock")
			}
		} else {
			if svc.stopLivenessCheck.Load() {
				// if locker is stop, resume it
				log.Logger.Debug("msg", "prometheus liveness check pass, resume locker")
				svc.stopLivenessCheck.Store(false)
				svc.retryLock.Reset(svc.elector.RetryPeriod())
			}
		}
	}
}

func (svc *Service) lockerRetry(ctx context.Context) {
	for {
		select {
		case <-svc.retryLock.C:
			if !svc.elector.IsLeader() {
				err := svc.elector.StartLeading(ctx)
				if err != nil {
					log.Logger.Debug("msg", "server keep trying get leader failed")
				} else {
					svc.keepAlive.Reset(svc.elector.HeartBeat())
				}
			}
		}
	}
}

func (svc *Service) lockerKeepAlive(ctx context.Context) {
	for {
		select {
		case <-svc.keepAlive.C:
			if svc.elector.IsLeader() {
				svc.elector.KeepAlive(ctx)
				log.Logger.Debug("msg", "server locker keep alive")
			}
		}
	}
}
