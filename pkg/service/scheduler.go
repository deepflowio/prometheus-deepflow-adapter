package service

import (
	"context"
	"prometheus-deepflow-adapter/pkg/log"
	"time"
)

func (svc *Service) prometheusLivenessCheck(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		elapsed := time.Since(time.Unix(0, svc.lastReceiveTime))
		if svc.elector.IsLeader() {
			if elapsed > svc.conf.PrometheusTimeout {
				// prometheus liveness check failed
				svc.stopLockerLivenessChecker <- true
				err := svc.elector.Release(ctx)
				if err != nil {
					log.Logger.Error("msg", "release elector locker failed", "err", err)
				}
				log.Logger.Info("msg", "prometheus liveness failed, stop trying locker")
			} else {
				log.Logger.Debug("msg", "prometheus liveness confirm")
			}
		} else {
			if svc.stopLivenessCheck.Load() {
				// if locker is stop, resume it
				log.Logger.Debug("msg", "prometheus liveness check pass, resume locker")
				svc.stopLockerLivenessChecker <- false
			}
		}
	}
}

func (svc *Service) lockerLivenessCheck(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case res := <-svc.stopLockerLivenessChecker:
			svc.stopLivenessCheck.Store(res)
		case <-ticker.C:
			if !svc.stopLivenessCheck.Load() {
				err := svc.elector.StartLeading(ctx)
				if err != nil {
					log.Logger.Debug("msg", "server keep trying get leader failed")
				}
			}
		}
	}
}
