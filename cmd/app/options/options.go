package options

import (
	"prometheus-deepflow-adapter/pkg/config"

	cliflag "k8s.io/component-base/cli/flag"
)

type Options struct {
}

func NewOptions() *Options {
	return &Options{}
}

func (a *Options) Flags(config config.Configuration) cliflag.NamedFlagSets {
	fs := cliflag.NamedFlagSets{}
	gfs := fs.FlagSet("generic")
	gfs.AddFlagSet(config.ToOptions())
	return fs
}
