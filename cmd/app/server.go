package app

import (
	"errors"
	"github.com/spf13/cobra"
	"prometheus_adapter/cmd/app/options"
	"prometheus_adapter/pkg/plugins"
)

func NewAdapterCommand() *cobra.Command {
	s := options.NewAdapterOptions()
	cmd := &cobra.Command{
		Use: "vela-core",
		Long: `The KubeVela controller manager is a daemon that embeds
the core control loops shipped with KubeVela`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(s)
		},
		SilenceUsage: true,
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags()
	for _, set := range namedFlagSets.FlagSets {
		fs.AddFlagSet(set)
	}

	return cmd
}

func run(s *options.AdapterOptions) error {
	switch {
	case s.K8S:
		prometheusAction, err := plugins.NewK8sClient()
		if err != nil {
			return err
		}
		prometheusAction.Run()
	case s.Etcd:
		if len(s.EtcdEndpoints) == 0 {
			return errors.New("etcd endpoints is null")
		}
		prometheusAction, err := plugins.NewEtcdClient(s.EtcdEndpoints)
		if err != nil {
			return err
		}
		prometheusAction.Run()
	default:
		return errors.New("unknown plugin")
	}

	return nil
}
