package config

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

type Configuration interface {
	ToOptions() *pflag.FlagSet
}

type Config struct {
	ElectionEnabled          bool          `mapstructure:"election-enabled"`
	TraceEnabled             bool          `mapstructure:"trace-enabled"`
	EventEnabled             bool          `mapstructure:"event-enabled"`
	ProfileEnabled           bool          `mapstructure:"profile-enabled"`
	PrometheusScrapeInterval time.Duration `mapstructure:"prometheus-scrape-interval"`

	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log-level"`

	Elector Elector `mapstructure:"elector"`

	// functional config
	RemoteWriteConfig RemoteWriteConfig `mapstructure:"remote-write"`
	// +WalConfig

	// debug-level config
	TraceConfig   TraceConfig   `mapstructure:"trace"`
	ProfileConfig ProfileConfig `mapstructure:"profile"`

	ExtraConfigs map[string]Configuration `mapstructure:"-"`
}

func NewConfig() *Config {
	cfg := &Config{
		Port:              80,
		LogLevel:          "info",
		RemoteWriteConfig: RemoteWriteConfig{},
		TraceConfig:       TraceConfig{},
		ProfileConfig:     ProfileConfig{},
	}
	return cfg
}

func (c *Config) ToOptions() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ExitOnError)

	fs.BoolVar(&c.ElectionEnabled, "election-enabled", true, "enable/disable election")
	fs.BoolVar(&c.TraceEnabled, "trace-enabled", false, "enable/disable distributed tracing")
	fs.BoolVar(&c.EventEnabled, "event-enabled", false, "enable/disable tracing event")
	fs.BoolVar(&c.ProfileEnabled, "profile-enabled", false, "enable/disable go profile")
	fs.DurationVar(&c.PrometheusScrapeInterval, "prometheus-scrape-interval", 10*time.Second, "timeout calculation for receive promtheus data")

	fs.IntVarP(&c.Port, "port", "p", 80, "http listen port")
	fs.StringVar(&c.LogLevel, "log-level", "info", "log level for adapter")
	fs.StringVar((*string)(&c.Elector), "elector", "k8s", "choose one election component")

	fs.AddFlagSet(c.RemoteWriteConfig.ToOptions())
	fs.AddFlagSet(c.TraceConfig.ToOptions())
	fs.AddFlagSet(c.ProfileConfig.ToOptions())

	c.ExtraConfigs = make(map[string]Configuration, len(extraConfigs))
	for k, configInit := range extraConfigs {
		c.ExtraConfigs[k] = configInit()
		fs.AddFlagSet(c.ExtraConfigs[k].ToOptions())
	}

	return fs
}

type TLSConfig struct {
	CAFile     string `mapstructure:"ca-file"`
	CertFile   string `mapstructure:"cert-file"`
	KeyFile    string `mapstructure:"key-file"`
	ServerName string `mapstructure:"server-name"`
}

type RemoteWriteConfig struct {
	Url       string        `mapstructure:"url"`
	Insecure  bool          `mapstructure:"insecure"`
	Timeout   time.Duration `mapstructure:"timeout"`
	TLSConfig TLSConfig     `mapstructure:"tls-config"`
}

func (r *RemoteWriteConfig) ToOptions() *pflag.FlagSet {
	fs := pflag.NewFlagSet("remote-write", pflag.ContinueOnError)
	fs.StringVar(&r.Url, "url", "", "remote write url")
	fs.BoolVar(&r.Insecure, "insecure", false, "insecure config for remote write")
	fs.DurationVar(&r.Timeout, "timeout", 10*time.Second, "remote write timeout")
	fs.StringVar(&r.TLSConfig.CAFile, "ca-file", "", "remote write https ca")
	fs.StringVar(&r.TLSConfig.CertFile, "cert-file", "", "remote write https cert file")
	fs.StringVar(&r.TLSConfig.KeyFile, "key-file", "", "remote write https key file")
	fs.StringVar(&r.TLSConfig.ServerName, "server-name", "", "remote write https server name")
	fs.VisitAll(func(f *pflag.Flag) {
		f.Name = fmt.Sprintf("%s-%s", "remote-write", f.Name)
	})
	return fs
}

type TraceConfig struct {
	ClientType ClientType `mapstructure:"client-type"`

	Endpoint  string        `mapstructure:"endpoint"`
	Insecure  bool          `mapstructure:"insecure"`
	Timeout   time.Duration `mapstructure:"timeout"`
	TLSConfig TLSConfig
}

func (t *TraceConfig) ToOptions() *pflag.FlagSet {
	fs := pflag.NewFlagSet("trace", pflag.ContinueOnError)
	fs.StringVar((*string)(&t.ClientType), "client-type", "http", "trace client type, http/grpc, default: http")
	fs.StringVar(&t.Endpoint, "endpoint", "", "trace backend endpoint")
	fs.BoolVar(&t.Insecure, "insecure", false, "trace endpoint insecure")
	fs.DurationVar(&t.Timeout, "timeout", 10*time.Second, "trace timeout")
	fs.StringVar(&t.TLSConfig.CAFile, "ca-file", "", "trace https ca file")
	fs.StringVar(&t.TLSConfig.CertFile, "cert-file", "", "trace https cert file")
	fs.StringVar(&t.TLSConfig.KeyFile, "key-file", "", "trace https key file")
	fs.StringVar(&t.TLSConfig.ServerName, "server-name", "", "trace https server name")
	fs.VisitAll(func(f *pflag.Flag) {
		f.Name = fmt.Sprintf("%s-%s", "trace", f.Name)
	})
	return fs
}

type ProfileConfig struct {
	Rate  int      `mapstructure:"rate"`
	Types []string `mapstructure:"types"`
}

func (p *ProfileConfig) ToOptions() *pflag.FlagSet {
	fs := pflag.NewFlagSet("profile", pflag.ContinueOnError)
	fs.IntVar(&p.Rate, "rate", 0, "profile rate")
	fs.StringSliceVar(&p.Types, "type", []string{}, "profile types")
	fs.VisitAll(func(f *pflag.Flag) {
		f.Name = fmt.Sprintf("%s-%s", "profile", f.Name)
	})
	return fs
}
