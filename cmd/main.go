package main

import (
	"os"
	"prometheus-deepflow-adapter/cmd/app"
)

func main() {
	command := app.NewAdapter()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
