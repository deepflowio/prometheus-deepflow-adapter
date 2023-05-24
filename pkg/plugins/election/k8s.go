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
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"prometheus-deepflow-adapter/pkg/config"
	"prometheus-deepflow-adapter/pkg/utils"
)

type k8sElector struct {
	uuid     string
	config   *K8SConfig
	client   *kubernetes.Clientset
	isLeader *atomic.Bool

	lock resourcelock.Interface
}

func Newk8sElector(config config.Configuration) (Election, error) {
	conf := config.(*K8SConfig)
	cfg, err := clientcmd.BuildConfigFromFlags("", conf.KubeConfig)
	if err != nil {
		return nil, err
	}
	k := &k8sElector{
		client:   kubernetes.NewForConfigOrDie(cfg),
		config:   conf,
		uuid:     uuid.NewString(),
		isLeader: &atomic.Bool{},
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Name:            utils.GetProcessName(),
		Lock:            k.lock,
		ReleaseOnCancel: true,
		LeaseDuration:   30 * time.Second,
		RenewDeadline:   k.config.HeartBeat * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				k.isLeader.Store(true)
			},
			OnStoppedLeading: func() {
				k.Release(context.Background())
			},
		},
	})
	return nil
}

func (k *k8sElector) Release(ctx context.Context) error {
	k.isLeader.Store(false)
	return nil
}

func (k *k8sElector) IsLeader() bool {
	return k.isLeader.Load()
}

func (k *k8sElector) KeepAlive(ctx context.Context) {
	// nothing, k8s lease will keep alive
}

type K8SConfig struct {
	KubeConfig string `mapstructure:"kube-config"`

	HeartBeat          time.Duration `mapstructure:"heartbeat"`
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
