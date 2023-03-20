package options

import cliflag "k8s.io/component-base/cli/flag"

type AdapterOptions struct {
	Etcd          bool
	EtcdEndpoints []string
	K8S           bool
	Kubeconfig    string
}

func NewAdapterOptions() *AdapterOptions {
	return &AdapterOptions{
		Etcd: false,
		K8S:  false,
	}
}

func (a *AdapterOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	gfs := fss.FlagSet("generic")
	gfs.BoolVar(&a.Etcd, "etcd", a.Etcd, "Enable Etcd Elections")
	gfs.BoolVar(&a.K8S, "k8s", a.K8S, "Enable Kubernetes Elections")
	gfs.StringArrayVar(&a.EtcdEndpoints, "endpoints", a.EtcdEndpoints, "Etcd Endpoints")
	gfs.StringVar(&a.Kubeconfig, "kubeconfig", a.Kubeconfig, "Kubeconfig file")

	return fss
}
