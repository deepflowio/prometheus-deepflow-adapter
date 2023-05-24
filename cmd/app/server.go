package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/spf13/cobra"

	"prometheus-deepflow-adapter/cmd/app/options"
	"prometheus-deepflow-adapter/pkg/config"
	plog "prometheus-deepflow-adapter/pkg/log"
	"prometheus-deepflow-adapter/pkg/service"
)

var k = koanf.New(".")
var conf = config.NewConfig()

func NewAdapter() *cobra.Command {
	s := options.NewOptions()
	cmd := &cobra.Command{
		Use:  "deepflow-adapter",
		Long: "deepflow adapter for prometheus remote write to deepflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(context.Background(), conf)
		},
		SilenceUsage: true,
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags(conf)
	for _, set := range namedFlagSets.FlagSets {
		fs.AddFlagSet(set)
	}
	fs.Parse(os.Args[1:])

	/*
		example config: k8s-kube-config
		parse in commandline: --k8s-kube-config
		parse in env: k8s.kube-config
		finally: env overwrite commandline args
	*/
	if err := k.Load(posflag.Provider(fs, ".", k), nil); err != nil {
		log.Fatal(err)
	}

	if err := k.Load(env.Provider("", ".", func(s string) string {
		return s
	}), nil); err != nil {
		log.Fatal(err)
	}

	if err := k.UnmarshalWithConf("", &conf, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		log.Fatal(err)
	}

	return cmd
}

func run(ctx context.Context, c *config.Config) error {
	plog.Logger = plog.NewLogger(c.LogLevel)
	httpService := service.NewService(c)
	go func() {
		plog.Logger.Info("msg", "http server start up")
		if err := httpService.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			plog.Logger.Error("msg", "http server start up error", "err", err)
		}
	}()

	gracefullyQuit := make(chan os.Signal, 1)
	signal.Notify(gracefullyQuit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-gracefullyQuit
	close(gracefullyQuit)

	ctx, done := context.WithTimeout(ctx, 10*time.Second)
	defer done()

	if err := httpService.Shutdown(ctx); err != nil {
		plog.Logger.Error("msg", "http server shutdown error", "err", err)
	}

	return nil
}
