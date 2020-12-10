package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/GoogleContainerTools/skaffold/hack/perf/config"
	"github.com/GoogleContainerTools/skaffold/hack/perf/metrics"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "file", "config.yaml", "path to config file")
}

func main() {
	if err := collectMetrics(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// TODO:
// Store skaffold events in a file ~/.skaffold/events
// After running the dev, changing the file, and killing it, get skaffold events
// Parse out times and send data to cloud monitoring

func collectMetrics() error {
	cfg, err := config.Get(configFile)
	if err != nil {
		return fmt.Errorf("getting config: %w", err)
	}
	for _, app := range cfg.Apps {
		metrics.InnerLoop(app)
	}
	return nil
}
