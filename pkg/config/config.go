package config

import (
	"flag"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const ENVMode = "MODE"
const DebugMode = "debug"

func Mode() string {
	mode := viper.GetString(ENVMode)
	if mode == "" {
		return DebugMode
	}

	return mode
}

var (
	Kubeconfig         string
	LeaseLockName      string
	LeaseLockNamespace string
	Id                 string
	RemoteUrl          string
	ListenPort         string
)

func init() {
	flag.StringVar(&Kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&Id, "id", uuid.New().String(), "the holder identity name")
	flag.StringVar(&LeaseLockName, "lease-lock-name", "", "the lease lock resource name")
	flag.StringVar(&LeaseLockNamespace, "lease-lock-namespace", "", "the lease lock resource namespace")
	flag.StringVar(&RemoteUrl, "remoteUrl", "", "remote write url")
	flag.StringVar(&ListenPort, "listenPort", ":8080", "listenPort=:8080")
	flag.Parse()
}
