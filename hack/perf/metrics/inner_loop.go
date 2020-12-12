package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleContainerTools/skaffold/hack/perf/config"
	"github.com/GoogleContainerTools/skaffold/hack/perf/events"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/sirupsen/logrus"
)

// InnerLoopMetrics collects metrics for the inner loop and exports them
// to Cloud Monitoring
func InnerLoopMetrics(ctx context.Context, app config.Application) error {
	ef, err := events.File()
	if err != nil {
		return fmt.Errorf("events file: %w", err)
	}
	logEntries, err := events.GetFromFile(ef)
	if err != nil {
		return fmt.Errorf("getting events from file: %w", err)
	}
	innerLoopMetrics := splitEntriesByDevLoop(logEntries)
	logrus.Infof("Inner loop metrics for this run: %v", innerLoopMetrics)
	return exportInnerLoopMetrics(ctx, innerLoopMetrics[1])
}

func splitEntriesByDevLoop(logEntries []proto.LogEntry) []innerLoopMetric {
	var ilms []innerLoopMetric

	var current innerLoopMetric
	var buildStartTime, deployStartTime, statusCheckStartTime time.Time
	for _, le := range logEntries {
		switch le.Event.GetEventType().(type) {
		case *proto.Event_MetaEvent:
			fmt.Println("metadata event logic not yet implemented")
		case *proto.Event_DevLoopEvent:
			// we have reached the end of a dev loop
			status := le.GetEvent().GetDevLoopEvent().GetStatus()
			if status == event.Succeeded {
				buildStartTime, deployStartTime, statusCheckStartTime = time.Time{}, time.Time{}, time.Time{}
				ilms = append(ilms, current)
			}
		case *proto.Event_BuildEvent:
			status := le.GetEvent().GetBuildEvent().GetStatus()
			unixTimestamp := time.Unix(le.GetTimestamp().AsTime().Unix(), 0)
			if status == event.InProgress && buildStartTime.IsZero() {
				buildStartTime = unixTimestamp
			} else if status == event.Complete {
				current.buildTime = unixTimestamp.Sub(buildStartTime).Seconds()
			}
		case *proto.Event_DeployEvent:
			status := le.GetEvent().GetDeployEvent().GetStatus()
			unixTimestamp := time.Unix(le.GetTimestamp().AsTime().Unix(), 0)
			if status == event.InProgress {
				deployStartTime = unixTimestamp
				// deploy is finished when it is marked "Complete"
			} else if status == event.Complete {
				current.deployTime = unixTimestamp.Sub(deployStartTime).Seconds()
			}
		case *proto.Event_StatusCheckEvent:
			status := le.GetEvent().GetStatusCheckEvent().GetStatus()
			unixTimestamp := time.Unix(le.GetTimestamp().AsTime().Unix(), 0)
			if status == event.Started {
				statusCheckStartTime = unixTimestamp
			} else if status == event.Succeeded {
				current.statusCheckTime = unixTimestamp.Sub(statusCheckStartTime).Seconds()
			}
		}
	}
	return ilms
}

type innerLoopMetric struct {
	buildTime       float64
	deployTime      float64
	statusCheckTime float64
}
