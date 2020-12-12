package metrics

import (
	"context"
	"fmt"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

const (
	customMetricName = "custom.googleapis.com/skaffold/dev"
)

var (
	buildLatencyS       = stats.Float64("repl/buildTime", "build time in seconds", "s")
	deployLatencyS      = stats.Float64("repl/deployTime", "deploy time in seconds", "s")
	statusCheckLatencyS = stats.Float64("repl/statusCheckTime", "status check time in seconds", "s")
	// this should equal build + deploy + status check time
	totalInnerLoopS = stats.Float64("repl/innerLoopTime", "inner loop time in seconds", "s")
	labels          string
	tmpFile         string
)

func exportInnerLoopMetrics(ctx context.Context, ilm innerLoopMetric) error {
	if err := registerViews(); err != nil {
		return fmt.Errorf("registering views: %w", err)
	}
	sd, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: projectID(),
		// ReportingInterval sets the frequency of reporting metrics
		// to stackdriver backend.
		ReportingInterval: 1 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("stackdriver new exporter: %w", err)
	}
	// Register it as a trace exporter
	trace.RegisterExporter(sd)

	if err := sd.StartMetricsExporter(); err != nil {
		return fmt.Errorf("starting metrics exporter: %w", err)
	}

	stats.Record(ctx, buildLatencyS.M(ilm.buildTime))
	stats.Record(ctx, deployLatencyS.M(ilm.deployTime))
	stats.Record(ctx, statusCheckLatencyS.M(ilm.statusCheckTime))
	stats.Record(ctx, totalInnerLoopS.M(ilm.buildTime+ilm.deployTime+ilm.statusCheckTime))

	time.Sleep(30 * time.Second)
	sd.Flush()
	sd.StopMetricsExporter()
	trace.UnregisterExporter(sd)

	return nil
}

func projectID() string {
	return "priya-wadhwa"
}

// Register the view. It is imperative that this step exists,
// otherwise recorded metrics will be dropped and never exported.
func registerViews() error {
	views := map[string]*stats.Float64Measure{
		"build":       buildLatencyS,
		"deploy":      deployLatencyS,
		"statusCheck": statusCheckLatencyS,
		"total":       totalInnerLoopS,
	}
	for name, measure := range views {
		v := &view.View{
			Name:        fmt.Sprintf("%s/%s", customMetricName, name),
			Measure:     measure,
			Aggregation: view.LastValue(),
		}
		if err := view.Register(v); err != nil {
			return fmt.Errorf("registering view: %w", err)
		}
	}
	return nil
}
