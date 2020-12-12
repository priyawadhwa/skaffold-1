package metrics

import (
	"reflect"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func TestSplitEntriesByDevLoop(t *testing.T) {
	expectedILMS := []innerLoopMetric{
		{
			buildTime:       20,
			deployTime:      10,
			statusCheckTime: 5,
		}, {
			buildTime:       10,
			deployTime:      20,
			statusCheckTime: 0,
		},
	}
	actualILMS := splitEntriesByDevLoop(getLogEntries())
	if !reflect.DeepEqual(expectedILMS, actualILMS) {
		t.Errorf("Expected result different from actual result. Expected: \n%v, \nActual: \n%v", expectedILMS, actualILMS)
	}
}

func getLogEntries() []proto.LogEntry {
	now := time.Now().Unix()

	firstIteration := []proto.LogEntry{
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 100},
			Event:     &proto.Event{EventType: &proto.Event_MetaEvent{MetaEvent: &proto.MetaEvent{Metadata: &proto.Metadata{Build: &proto.BuildMetadata{NumberOfArtifacts: 1}}}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 100},
			Event:     &proto.Event{EventType: &proto.Event_BuildEvent{BuildEvent: &proto.BuildEvent{Status: event.InProgress}}},
		}, {
			Timestamp: &timestamp.Timestamp{Seconds: now - 80},
			Event:     &proto.Event{EventType: &proto.Event_BuildEvent{BuildEvent: &proto.BuildEvent{Status: event.Complete}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 80},
			Event:     &proto.Event{EventType: &proto.Event_DeployEvent{DeployEvent: &proto.DeployEvent{Status: event.InProgress}}},
		}, {
			Timestamp: &timestamp.Timestamp{Seconds: now - 70},
			Event:     &proto.Event{EventType: &proto.Event_DeployEvent{DeployEvent: &proto.DeployEvent{Status: event.Complete}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 70},
			Event:     &proto.Event{EventType: &proto.Event_StatusCheckEvent{StatusCheckEvent: &proto.StatusCheckEvent{Status: event.Started}}},
		}, {
			Timestamp: &timestamp.Timestamp{Seconds: now - 65},
			Event:     &proto.Event{EventType: &proto.Event_StatusCheckEvent{StatusCheckEvent: &proto.StatusCheckEvent{Status: event.Succeeded}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 65},
			Event:     &proto.Event{EventType: &proto.Event_DevLoopEvent{DevLoopEvent: &proto.DevLoopEvent{Status: event.Succeeded}}},
		},
	}

	secondIteration := []proto.LogEntry{
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 100},
			Event:     &proto.Event{EventType: &proto.Event_MetaEvent{MetaEvent: &proto.MetaEvent{Metadata: &proto.Metadata{Build: &proto.BuildMetadata{NumberOfArtifacts: 2}}}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 65},
			Event:     &proto.Event{EventType: &proto.Event_DevLoopEvent{DevLoopEvent: &proto.DevLoopEvent{Status: event.InProgress}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 60},
			Event:     &proto.Event{EventType: &proto.Event_BuildEvent{BuildEvent: &proto.BuildEvent{Status: event.InProgress}}},
		}, {
			Timestamp: &timestamp.Timestamp{Seconds: now - 55},
			Event:     &proto.Event{EventType: &proto.Event_BuildEvent{BuildEvent: &proto.BuildEvent{Status: event.Complete}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 55},
			Event:     &proto.Event{EventType: &proto.Event_BuildEvent{BuildEvent: &proto.BuildEvent{Status: event.InProgress}}},
		}, {
			Timestamp: &timestamp.Timestamp{Seconds: now - 50},
			Event:     &proto.Event{EventType: &proto.Event_BuildEvent{BuildEvent: &proto.BuildEvent{Status: event.Complete}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 50},
			Event:     &proto.Event{EventType: &proto.Event_DeployEvent{DeployEvent: &proto.DeployEvent{Status: event.InProgress}}},
		}, {
			Timestamp: &timestamp.Timestamp{Seconds: now - 30},
			Event:     &proto.Event{EventType: &proto.Event_DeployEvent{DeployEvent: &proto.DeployEvent{Status: event.Complete}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 20},
			Event:     &proto.Event{EventType: &proto.Event_StatusCheckEvent{StatusCheckEvent: &proto.StatusCheckEvent{Status: event.Started}}},
		}, {
			Timestamp: &timestamp.Timestamp{Seconds: now - 20},
			Event:     &proto.Event{EventType: &proto.Event_StatusCheckEvent{StatusCheckEvent: &proto.StatusCheckEvent{Status: event.Succeeded}}},
		},
		{
			Timestamp: &timestamp.Timestamp{Seconds: now - 15},
			Event:     &proto.Event{EventType: &proto.Event_DevLoopEvent{DevLoopEvent: &proto.DevLoopEvent{Status: event.Succeeded}}},
		},
	}

	return append(firstIteration, secondIteration...)
}
