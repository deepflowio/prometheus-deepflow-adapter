package plugins

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"log"
	"net/http"
	"prometheus_adapter/pkg/config"
	"prometheus_adapter/pkg/service"
)

type etcdClient struct {
	client *clientv3.Client
}

func NewEtcdClient(endpoint []string) (PrometheusAction, error) {
	endpoints := endpoint
	client, err := clientv3.New(clientv3.Config{Endpoints: endpoints})
	if err != nil {
		return nil, err
	}
	return etcdClient{
		client: client,
	}, nil
}

func (e etcdClient) Run() {
	session, err := concurrency.NewSession(e.client)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	locker := concurrency.NewMutex(session, "/p8s-lock")
	err = locker.TryLock(context.Background())
	if err != nil {
		fmt.Println("lock failed", err)
		return
	}

	router := service.NewEngine()
	srv := &http.Server{
		Addr:    config.ListenPort,
		Handler: router,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		locker.Unlock(context.Background())
		log.Fatalf("listen: %s\n", err)
	}
}
