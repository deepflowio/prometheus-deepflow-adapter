package election

import (
	"context"
	"fmt"
	"prometheus-deepflow-adapter/pkg/config"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type etcdElector struct {
	config   *EtcdConfig
	client   *clientv3.Client
	isLeader *atomic.Bool

	mutex   *concurrency.Mutex
	session *concurrency.Session
}

// implement etcd distribution locker
func NewEtcdElector(config config.Configuration) (Election, error) {
	conf := config.(*EtcdConfig)
	client, err := clientv3.New(clientv3.Config{Endpoints: conf.Endpoints})
	if err != nil {
		return nil, err
	}
	return &etcdElector{
		client:   client,
		config:   conf,
		isLeader: &atomic.Bool{},
	}, nil
}

func (e *etcdElector) StartLeading(ctx context.Context) error {
	var err error
	e.session, err = concurrency.NewSession(e.client,
		concurrency.WithContext(ctx),
		concurrency.WithTTL(int(e.config.HeartBeat)),
	)
	if err != nil {
		return err
	}
	// defer session.Close()

	e.mutex = concurrency.NewMutex(e.session, e.config.Key)
	err = e.mutex.TryLock(ctx)
	if err != nil {
		e.isLeader.Store(false)
		e.session.Close()
		return err
	} else {
		// close Session after Release
		e.isLeader.Store(true)
		return nil
	}
}

func (e *etcdElector) Release(ctx context.Context) error {
	if !e.IsLeader() {
		return nil
	}
	err := e.mutex.Unlock(ctx)
	if err != nil {
		return err
	}
	e.isLeader.Store(false)
	return e.session.Close()
}

func (e *etcdElector) IsLeader() bool {
	return e.isLeader.Load()
}

func (e *etcdElector) KeepAlive(ctx context.Context) {
	// nothing, etcd concurrency will keep alive
}

type EtcdConfig struct {
	Endpoints []string      `mapstructure:"endpoints"`
	HeartBeat time.Duration `mapstructure:"heartbeat"`
	Key       string        `mapstructure:"key"`
}

func NewEtcdConfig() config.Configuration {
	return &EtcdConfig{}
}

func (e *EtcdConfig) ToOptions() *pflag.FlagSet {
	fs := pflag.NewFlagSet("etcd", pflag.ContinueOnError)
	fs.StringSliceVar(&e.Endpoints, "endpoints", nil, "etcd endpoints")
	fs.StringVar(&e.Key, "key", "/p8s-df-adapter-lock", "etcd election keys")
	fs.DurationVar(&e.HeartBeat, "etcd-heartbeat", 15*time.Second, "lock heartbeat interval")
	fs.VisitAll(func(f *pflag.Flag) {
		f.Name = fmt.Sprintf("%s-%s", "etcd", f.Name)
	})
	return fs
}

func init() {
	config.RegisterConfig(string(config.Etcd), NewEtcdConfig)
	RegisterElector(config.Etcd, NewEtcdElector)
}
