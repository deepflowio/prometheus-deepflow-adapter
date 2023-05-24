package config

type Elector string

const (
	K8S       Elector = "k8s"
	Etcd      Elector = "etcd"
	Redis     Elector = "redis"
	Consul    Elector = "consul"
	Zookeeper Elector = "zookeeper"

	// not implement yet
	Others Elector = "unknown"
)

type ClientType string

const (
	http ClientType = "http"
	grpc ClientType = "grpc"
)
