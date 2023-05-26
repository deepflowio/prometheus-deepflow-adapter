package election

import (
	"context"
	"fmt"
	"prometheus-deepflow-adapter/pkg/config"
	"prometheus-deepflow-adapter/pkg/log"
	"time"
)

/*
    This election is not common election. Currently, it will try to distribution lock, then it has permission to do remote write.

	election status:
    election(lock success)
     ┌───────────┐   ┌──────────┐   ┌────────────┐   ┌────────────┐   ┌────────────┐
     │  pending  ├──►│  locked  ├──►│ keep alive ├──►│  released  │──►│  try lock  │
     └───────────┘   └──────────┘   └────────────┘   └────────────┘   └────────────┘
	 why released:
	 1. prometheus replica-x crashed and send nothing(lastReceiveTime > [config] prometheus timeout)
	 2. adapter crashed

    election(lock failed)
	 ┌───────────┐   ┌──────────┐   ┌────────────┐
     │  pending  ├──►│  failed  ├──►│  try lock  │
     └───────────┘   └──────────┘   └────────────┘
	 which adapter should try lock:
	 prometheus replica-x keep sending data(lastReceiveTime < [config] prometheus timeout)
*/

type Election interface {
	// requirement:
	// non-block implements, when try lock failed, StartLeading should return immediately
	// initialize `client` only one time, StartLeading will keep trying when current server is not leader, until it get lock.
	StartLeading(context.Context) error
	Release(context.Context) error
	IsLeader() bool
	KeepAlive(ctx context.Context)
	RetryPeriod() time.Duration
	HeartBeat() time.Duration
}

type electorConstructor func(config.Configuration) (Election, error)

var electorComponents = map[config.Elector]electorConstructor{}

func RegisterElector(name config.Elector, f electorConstructor) {
	electorComponents[name] = f
}

func StartElection(conf *config.Config) Election {
	electorFunc := electorComponents[conf.Elector]
	if electorFunc == nil {
		log.Logger.Error("msg", "get elector failed", "elector", conf.Elector)
		return nil
	}

	elector, err := electorFunc(conf.ExtraConfigs[string(conf.Elector)])
	if err != nil {
		log.Logger.Error("msg", "call elector constructor function failed", "elector", conf.Elector, "err", err)
		return nil
	}

	log.Logger.Info("msg", fmt.Sprintf("%selector start election now", conf.Elector))

	err = elector.StartLeading(context.Background())
	if err != nil {
		log.Logger.Info("msg", "try get leader lock failed, server is not leader", "elector", conf.Elector, "err", err)
	}
	return elector
}
