package election

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"prometheus-deepflow-adapter/pkg/config"
	"prometheus-deepflow-adapter/pkg/log"
	"prometheus-deepflow-adapter/pkg/utils"
)

type k8sElector struct {
	uuid     string
	config   *K8SConfig
	client   *kubernetes.Clientset
	isLeader *atomic.Bool
	done     context.CancelFunc

	lock          resourcelock.Interface
	leaseDuration time.Duration
}

func Newk8sElector(config config.Configuration) (Election, error) {
	conf := config.(*K8SConfig)
	var err error
	var cfg *rest.Config
	if conf.KubeConfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", conf.KubeConfig)
		if err != nil {
			return nil, err
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	k := &k8sElector{
		client:        kubernetes.NewForConfigOrDie(cfg),
		config:        conf,
		uuid:          uuid.NewString(),
		isLeader:      &atomic.Bool{},
		leaseDuration: 30 * time.Second,
	}

	k.lock = &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      k.config.LeaseLockName,
			Namespace: k.config.LeaseLockNamespace,
		},
		Client: k.client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: k.uuid,
		},
	}
	return k, nil
}

func (k *k8sElector) StartLeading(ctx context.Context) error {
	ctx, k.done = context.WithCancel(ctx)
	// here are 2 ways to make election non-block:
	// 1. sever become leader: return immediately, keep it block in a goroutine before Release()
	// 2. server is not leader: try lock and return after timeout
	electionTimeout := time.NewTicker(k.leaseDuration + 10*time.Second)
	startLeading := make(chan struct{})
	go func(c context.Context) {
		leaderelection.RunOrDie(c, leaderelection.LeaderElectionConfig{
			Name:            utils.GetProcessName(),
			Lock:            k.lock,
			ReleaseOnCancel: true,
			LeaseDuration:   k.leaseDuration,
			RenewDeadline:   k.config.HeartBeat,
			RetryPeriod:     k.config.RetryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					k.isLeader.Store(true)
					startLeading <- struct{}{}
				},
				OnStoppedLeading: func() {
					k.isLeader.Store(false)
				},
			},
		})
	}(ctx)

	select {
	case <-electionTimeout.C:
		// try lock failed, server is not leader and end election, avoid goroutine overflow
		k.done()
		log.Logger.Debug("msg", "server election timeout, server is not leader", "uuid", k.uuid, "elector", "k8s")
		return nil
	case <-startLeading:
		// election success, return immediately for non-block election
		electionTimeout.Stop()
		log.Logger.Debug("msg", "server become leader now, start data handlding", "uuid", k.uuid, "elector", "k8s")
		return nil
	}
}

func (k *k8sElector) Release(ctx context.Context) error {
	if k.IsLeader() {
		k.done()
	}
	return nil
}

func (k *k8sElector) IsLeader() bool {
	return k.isLeader.Load()
}

func (k *k8sElector) RetryPeriod() time.Duration {
	return k.config.RetryPeriod
}

func (k *k8sElector) HeartBeat() time.Duration {
	return k.config.HeartBeat
}

func (k *k8sElector) KeepAlive(ctx context.Context) {
	// nothing, k8s lease will keep alive
}

type K8SConfig struct {
	KubeConfig         string        `mapstructure:"kube-config"`
	HeartBeat          time.Duration `mapstructure:"heartbeat"`
	RetryPeriod        time.Duration `mapstructure:"retry-period"`
	LeaseLockName      string        `mapstructure:"lease-lock-name"`
	LeaseLockNamespace string        `mapstructure:"lease-lock-namespace"`
}

func NewK8SConfig() config.Configuration {
	return &K8SConfig{}
}

func (k *K8SConfig) ToOptions() *pflag.FlagSet {
	fs := pflag.NewFlagSet("k8s", pflag.ContinueOnError)
	fs.StringVar(&k.KubeConfig, "kube-config", "", "kubernetes config file")
	fs.DurationVar(&k.HeartBeat, "heartbeat", 15*time.Second, "lock heartbeat interval")
	fs.DurationVar(&k.RetryPeriod, "retry-period", 10*time.Second, "lock retry interval")
	fs.StringVar(&k.LeaseLockName, "lease-lock-name", "p8s-df-adapter-lock", "kubernetes lease lock name")
	fs.StringVar(&k.LeaseLockNamespace, "lease-lock-namespace", "default", "kubernetes lease lock namespace")
	fs.VisitAll(func(f *pflag.Flag) {
		f.Name = fmt.Sprintf("%s-%s", "k8s", f.Name)
	})
	return fs
}

func init() {
	config.RegisterConfig(string(config.K8S), NewK8SConfig)
	RegisterElector(config.K8S, Newk8sElector)
}
