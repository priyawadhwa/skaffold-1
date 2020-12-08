package main

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/skaffold/hack/perf/events"
	"github.com/pkg/errors"
)

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
	events, err := events.Get()
	if err != nil {
		return errors.Wrap(err, "getting events")
	}
	fmt.Println(events)
	return nil
}
