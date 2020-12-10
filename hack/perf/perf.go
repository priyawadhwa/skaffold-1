package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/GoogleContainerTools/skaffold/hack/perf/config"
	"github.com/GoogleContainerTools/skaffold/hack/perf/skaffold"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "file", "config.yaml", "path to config file")
}

func main() {
	if err := collectMetrics(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// TODO:
// Store skaffold events in a file ~/.skaffold/events
// After running the dev, changing the file, and killing it, get skaffold events
// Parse out times and send data to cloud monitoring

func collectMetrics(ctx context.Context) error {
	cfg, err := config.Get(configFile)
	if err != nil {
		return fmt.Errorf("getting config: %w", err)
	}
	for _, app := range cfg.Apps {
		if err := skaffold.Dev(ctx, app); err != nil {
			fmt.Printf("%v\n", err)
		}
	}
	return nil
}
