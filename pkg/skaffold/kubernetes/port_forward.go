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

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
)

// PortForwarder is responsible for selecting pods satisfying a certain condition and port-forwarding the exposed
// container ports within those pods. It also tracks and manages the port-forward connections.
type PortForwarder struct {
	Forwarder

	output      io.Writer
	podSelector PodSelector
	namespaces  []string
	label       string

	// forwardedPods is a map of portForwardEntry.key() (string) -> portForwardEntry
	forwardedPods map[string]*portForwardEntry

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts *sync.Map
}

type portForwardEntry struct {
	resourceVersion int
	resource        latest.PortForwardResource
	podName         string
	containerName   string
	portName        string
	localPort       int32

	cancel context.CancelFunc
}

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Forward(context.Context, *portForwardEntry) error
	Terminate(*portForwardEntry)
}

type kubectlForwarder struct{}

var (
	// For testing
	retrieveAvailablePort = util.GetAvailablePort
)

// Forward port-forwards a pod using kubectl port-forward
// It returns an error only if the process fails or was terminated by a signal other than SIGTERM
func (*kubectlForwarder) Forward(parentCtx context.Context, pfe *portForwardEntry) error {
	logrus.Debugf("Port forwarding %s", pfe)

	ctx, cancel := context.WithCancel(parentCtx)
	pfe.cancel = cancel

	cmd := exec.CommandContext(ctx, "kubectl", "port-forward", fmt.Sprintf("%s/%s", pfe.resource.Type, pfe.resource.Name), fmt.Sprintf("%d:%d", pfe.localPort, pfe.resource.Port), "--namespace", pfe.resource.Namespace)
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Start(); err != nil {
		if errors.Cause(err) == context.Canceled {
			return nil
		}
		return errors.Wrapf(err, "port forwarding %s/%s, port: %d to local port: %d, err: %s", pfe.resource.Type, pfe.resource.Name, pfe.resource.Port, pfe.localPort, buf.String())
	}

	event.PortForwarded(pfe.localPort, pfe.resource.Port, pfe.podName, pfe.containerName, pfe.resource.Namespace, pfe.portName, pfe.resource.Type, pfe.resource.Name)
	go cmd.Wait()
	return portForwardSuccessful(pfe.localPort)
}

func portForwardSuccessful(port int32) error {
	_, err := http.Get(fmt.Sprintf("http://%s:%d", util.Loopback, port))
	return err
}

// Terminate terminates an existing kubectl port-forward command using SIGTERM
func (*kubectlForwarder) Terminate(p *portForwardEntry) {
	logrus.Debugf("Terminating port-forward %s", p)

	if p.cancel != nil {
		p.cancel()
	}
}

// NewPortForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewPortForwarder(out io.Writer, podSelector PodSelector, namespaces []string, label string) *PortForwarder {
	return &PortForwarder{
		Forwarder:      &kubectlForwarder{},
		output:         out,
		podSelector:    podSelector,
		namespaces:     namespaces,
		forwardedPods:  make(map[string]*portForwardEntry),
		forwardedPorts: &sync.Map{},
		label:          label,
	}
}

// Stop terminates all kubectl port-forward commands.
func (p *PortForwarder) Stop() {
	for _, entry := range p.forwardedPods {
		p.Terminate(entry)
	}
}

// Start begins a pod watcher that port forwards any pods involving containers with exposed ports.
// TODO(r2d4): merge this event loop with pod watcher from log writer
func (p *PortForwarder) Start(ctx context.Context) error {
	aggregate := make(chan watch.Event)
	stopWatchers, err := AggregatePodWatcher(p.namespaces, aggregate)
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
	if err := p.portForwardResources(ctx, serviceResources); err != nil {
		logrus.Warnf("error port forwarding resources: %v", err)
	}

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
			resourceVersion, err := strconv.Atoi(pod.ResourceVersion)
			if err != nil {
				return errors.Wrap(err, "converting resource version to integer")
			}
			entry, err := p.getCurrentEntry(resource, resourceVersion)
			if err != nil {
				return errors.Wrap(err, "failed to get port forward entry")
			}
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

func (p *PortForwarder) getCurrentEntry(resource latest.PortForwardResource, resourceVersion int) (*portForwardEntry, error) {
	// determine if we have seen this before
	entry := &portForwardEntry{
		resourceVersion: resourceVersion,
		resource:        resource,
	}
	// If we have, return the current entry
	oldEntry, ok := p.forwardedPods[entry.key()]
	if ok {
		entry.localPort = oldEntry.localPort
		return entry, nil
	}

	// retrieve an open port on the host
	entry.localPort = int32(retrieveAvailablePort(int(resource.Port), p.forwardedPorts))
	return entry, nil
}

func (p *PortForwarder) forward(ctx context.Context, entry *portForwardEntry) error {
	if prevEntry, ok := p.forwardedPods[entry.key()]; ok {
		// Check if this is a new generation of pod
		if entry.resourceVersion > prevEntry.resourceVersion {
			p.Terminate(prevEntry)
		}
	}
	color.Default.Fprintln(p.output, fmt.Sprintf("Port Forwarding %s/%s %d -> %d", entry.resource.Type, entry.resource.Name, entry.resource.Port, entry.localPort))
	p.forwardedPods[entry.key()] = entry
	err := wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
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

// Key is an identifier for the lock on a port during the skaffold dev cycle.
func (p *portForwardEntry) key() string {
	return fmt.Sprintf("%s-%s-%s-%d", p.resource.Type, p.resource.Name, p.resource.Namespace, p.resource.Port)
}

// String is a utility function that returns the port forward entry as a user-readable string
func (p *portForwardEntry) String() string {
	return fmt.Sprintf("%s/%s/%s:%d", p.resource.Type, p.resource.Name, p.resource.Namespace, p.resource.Port)
}
