/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package portforward

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
)

var (
	// For testing
	forwardingPollTime = time.Minute
)

// PortForwarder is responsible for selecting pods satisfying a certain condition and port-forwarding the exposed
// container ports within those pods. It also tracks and manages the port-forward connections.
type PortForwarder struct {
	Forwarder

	output      io.Writer
	podSelector kubernetes.PodSelector
	namespaces  []string
	label       string

	// forwardedResources is a map of portForwardEntry.key() (string) -> portForwardEntry
	forwardedResources map[string]*portForwardEntry

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts *sync.Map
}

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Forward(context.Context, *portForwardEntry) error
	Terminate(*portForwardEntry)
}

var (
	// For testing
	retrieveAvailablePort = util.GetAvailablePort
)

// NewPortForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewPortForwarder(out io.Writer, podSelector kubernetes.PodSelector, namespaces []string, label string) *PortForwarder {
	return &PortForwarder{
		Forwarder:          &kubectlForwarder{},
		output:             out,
		podSelector:        podSelector,
		namespaces:         namespaces,
		forwardedResources: make(map[string]*portForwardEntry),
		forwardedPorts:     &sync.Map{},
		label:              label,
	}
}

// Stop terminates all kubectl port-forward commands.
func (p *PortForwarder) Stop() {
	for _, entry := range p.forwardedResources {
		p.Terminate(entry)
	}
}

// Start begins a pod watcher that port forwards any pods involving containers with exposed ports.
// TODO(r2d4): merge this event loop with pod watcher from log writer
func (p *PortForwarder) Start(ctx context.Context) error {
	aggregate := make(chan watch.Event)
	stopWatchers, err := kubernetes.AggregatePodWatcher(p.namespaces, aggregate)
	if err != nil {
		stopWatchers()
		return errors.Wrap(err, "initializing pod watcher")
	}

	go func() {
		defer stopWatchers()

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-aggregate:
				if !ok {
					return
				}

				// If the event's type is "ERROR", warn and continue.
				if evt.Type == watch.Error {
					logrus.Warnf("got unexpected event of type %s", evt.Type)
					continue
				}
				// Grab the pod from the event.
				pod, ok := evt.Object.(*v1.Pod)
				if !ok {
					continue
				}
				// If the event's type is "DELETED", continue.
				if evt.Type == watch.Deleted {
					continue
				}

				// At this point, we know the event's type is "ADDED" or "MODIFIED".
				// We must take both types into account as it is possible for the pod to have become ready for port-forwarding before we established the watch.
				if p.podSelector.Select(pod) && pod.Status.Phase == v1.PodRunning && pod.DeletionTimestamp == nil {
					if err := p.portForwardPod(ctx, pod); err != nil {
						logrus.Warnf("port forwarding pod failed: %s", err)
					}
				}
			}
		}
	}()

	serviceResources, err := RetrieveServicesResources(p.label)
	if err != nil {
		logrus.Warnf("error retrieving service resources, will not port forward: %v", err)
	}

	p.portForwardResources(ctx, serviceResources)
	return nil
}

func (p *PortForwarder) portForwardPod(ctx context.Context, pod *v1.Pod) error {
	for _, c := range pod.Spec.Containers {
		for _, port := range c.Ports {
			// get current entry for this container
			resource := latest.PortForwardResource{
				Type:      "pod",
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Port:      port.ContainerPort,
			}

			entry := p.getCurrentEntry(resource)
			if err := updateEntryWithPodDetails(pod, resource, entry); err != nil {
				return errors.Wrap(err, "updating port forward entry with pod details")
			}
			if entry.resource.Port != entry.localPort {
				color.Yellow.Fprintf(p.output, "Forwarding container %s to local port %d.\n", c.Name, entry.localPort)
			}
			if err := p.forward(ctx, entry); err != nil {
				return errors.Wrap(err, "failed to forward port")
			}
		}
	}
	return nil
}

func updateEntryWithPodDetails(pod *v1.Pod, resource latest.PortForwardResource, entry *portForwardEntry) error {
	resourceVersion, err := strconv.Atoi(pod.ResourceVersion)
	if err != nil {
		return errors.Wrap(err, "converting resource version to integer")
	}
	entry.resourceVersion = resourceVersion
	entry.podName = pod.Name
	// determine the container name and port name for this entry
	containerName, portName, err := retrieveContainerNameAndPortNameFromPod(pod, resource.Port)
	if err != nil {
		return errors.Wrapf(err, "retrieving container and port name for %s/%s", resource.Type, resource.Name)
	}
	entry.containerName = containerName
	entry.portName = portName
	return nil
}

func (p *PortForwarder) getCurrentEntry(resource latest.PortForwardResource) *portForwardEntry {
	// determine if we have seen this before
	entry := &portForwardEntry{
		resource: resource,
	}
	// If we have, return the current entry
	oldEntry, ok := p.forwardedResources[entry.key()]
	if ok {
		entry.localPort = oldEntry.localPort
		return entry
	}

	// retrieve an open port on the host
	entry.localPort = int32(retrieveAvailablePort(int(resource.Port), p.forwardedPorts))
	return entry
}

func (p *PortForwarder) forward(ctx context.Context, entry *portForwardEntry) error {
	if prevEntry, ok := p.forwardedResources[entry.key()]; ok {
		// Check if this is a new generation of pod
		if entry.resourceVersion > prevEntry.resourceVersion {
			p.Terminate(prevEntry)
		}
	}
	color.Default.Fprintln(p.output, fmt.Sprintf("Port Forwarding %s/%s %d -> %d", entry.resource.Type, entry.resource.Name, entry.resource.Port, entry.localPort))
	p.forwardedResources[entry.key()] = entry
	err := wait.PollImmediate(time.Second, forwardingPollTime, func() (bool, error) {
		if err := p.Forward(ctx, entry); err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
}
