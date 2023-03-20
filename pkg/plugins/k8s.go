package plugins

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"log"
	"net/http"
	"os"
	"prometheus_adapter/pkg/config"
	"prometheus_adapter/pkg/service"
	"time"
)

type k8sClient struct {
	client *kubernetes.Clientset
}

func NewK8sClient() (PrometheusAction, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return k8sClient{
		client: client,
	}, nil

}

func (k8s k8sClient) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router := service.NewEngine()

	srv := &http.Server{
		Addr:    config.ListenPort,
		Handler: router,
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      config.LeaseLockName,
			Namespace: config.LeaseLockNamespace,
		},
		Client: k8s.client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: config.Id,
		},
	}

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("listen: %s\n", err)
				}
			},
			OnStoppedLeading: func() {
				if err := srv.Shutdown(ctx); err != nil {
					log.Println(err)
				}
				log.Printf("leader lost: %s", config.Id)
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				if identity == config.Id {
					return
				}
				log.Printf("new leader elected: %s", identity)
			},
		},
	})
}
