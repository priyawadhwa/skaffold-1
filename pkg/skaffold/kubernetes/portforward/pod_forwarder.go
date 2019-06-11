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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// AutomaticPodForwarder is responsible for selecting pods satisfying a certain condition and port-forwarding the exposed
// container ports within those pods. It also tracks and manages the port-forward connections.
type AutomaticPodForwarder struct {
	*kubectlForwarder

	output      io.Writer
	podSelector kubernetes.PodSelector
	namespaces  []string

	// forwardedPods is a map of portForwardEntry.podKey() (string) -> portForwardEntry
	forwardedPods map[string]*portForwardEntry

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts *sync.Map
}

// NewAutomaticPodForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewAutomaticPodForwarder(out io.Writer, podSelector kubernetes.PodSelector, namespaces []string, forwardedPorts *sync.Map) *AutomaticPodForwarder {
	return &AutomaticPodForwarder{
		kubectlForwarder: &kubectlForwarder{},
		output:           out,
		podSelector:      podSelector,
		namespaces:       namespaces,
		forwardedPods:    make(map[string]*portForwardEntry),
		forwardedPorts:   forwardedPorts,
	}
}

// Stop terminates all kubectl port-forward commands.
func (p *AutomaticPodForwarder) Stop() {
	for _, entry := range p.forwardedPods {
		p.Terminate(entry)
	}
}

func (p *AutomaticPodForwarder) Start(ctx context.Context) error {
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

	return nil
}

func (p *AutomaticPodForwarder) portForwardPod(ctx context.Context, pod *v1.Pod) error {
	for _, c := range pod.Spec.Containers {
		for _, port := range c.Ports {
			// get current entry for this container
			resource := latest.PortForwardResource{
				Type:      "pod",
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Port:      port.ContainerPort,
			}

			entry, err := p.getAutomaticPodForwardingEntry(pod, resource)
			if err != nil {
				return errors.Wrap(err, "getting automatic pod forwarding entry")
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

// forward the portForwardEntry
func (p *AutomaticPodForwarder) forward(ctx context.Context, entry *portForwardEntry) error {
	if prevEntry, ok := p.forwardedPods[entry.podKey()]; ok {
		// Check if this is a new generation of pod
		if entry.resourceVersion > prevEntry.resourceVersion {
			p.Terminate(prevEntry)
		}
	}
	p.forwardedPods[entry.podKey()] = entry
	color.Default.Fprintln(p.output, fmt.Sprintf("Port Forwarding %s/%s %d -> %d", entry.resource.Type, entry.resource.Name, entry.resource.Port, entry.localPort))
	if err := p.Forward(ctx, entry); err != nil {
		return errors.Wrap(err, "port forwarding failed")
	}
	return nil
}

func (p *AutomaticPodForwarder) getAutomaticPodForwardingEntry(pod *v1.Pod, resource latest.PortForwardResource) (*portForwardEntry, error) {
	entry := &portForwardEntry{
		resource: resource,
	}
	resourceVersion, err := strconv.Atoi(pod.ResourceVersion)
	if err != nil {
		return nil, errors.Wrap(err, "converting resource version to integer")
	}
	entry.resourceVersion = resourceVersion
	entry.podName = pod.Name
	// determine the container name and port name for this entry
	containerName, portName, err := retrieveContainerNameAndPortNameFromPod(pod, resource.Port)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving container and port name for %s/%s", resource.Type, resource.Name)
	}
	entry.containerName = containerName
	entry.portName = portName

	// If we have, return the current entry
	oldEntry, ok := p.forwardedPods[entry.podKey()]
	if ok {
		entry.localPort = oldEntry.localPort
		return entry, nil
	}

	// retrieve an open port on the host
	entry.localPort = int32(retrieveAvailablePort(int(resource.Port), p.forwardedPorts))

	return entry, nil
}

// retrieveContainerNameAndPortNameFromPod returns the container name and port name for a given port and pod
func retrieveContainerNameAndPortNameFromPod(pod *v1.Pod, port int32) (string, string, error) {
	for _, c := range pod.Spec.InitContainers {
		for _, p := range c.Ports {
			if p.ContainerPort == port {
				return c.Name, p.Name, nil
			}
		}
	}
	for _, c := range pod.Spec.Containers {
		for _, p := range c.Ports {
			if p.ContainerPort == port {
				return c.Name, p.Name, nil
			}
		}
	}
	return "", "", fmt.Errorf("pod %s does not expose port %d", pod.Name, port)
}
