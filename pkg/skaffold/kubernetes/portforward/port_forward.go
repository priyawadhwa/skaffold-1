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
	"sync"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	// For testing
	forwardingPollTime = time.Minute
)

// PortForwarder is responsible for selecting pods satisfying a certain condition and port-forwarding the exposed
// container ports within those pods. It also tracks and manages the port-forward connections.
type PortForwarder struct {
	*kubectlForwarder

	output      io.Writer
	podSelector kubernetes.PodSelector
	namespaces  []string
	label       string

	// forwardedResources is a map of portForwardEntry.key() (string) -> portForwardEntry
	forwardedResources map[string]*portForwardEntry

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts *sync.Map
}

var (
	// For testing
	retrieveAvailablePort = util.GetAvailablePort
)

// NewPortForwarder returns a struct that tracks and port-forwards pods as they are created and modified
func NewPortForwarder(out io.Writer, podSelector kubernetes.PodSelector, namespaces []string, label string) *PortForwarder {
	return &PortForwarder{
		kubectlForwarder:   &kubectlForwarder{},
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
	serviceResources, err := RetrieveServicesResources(p.label)
	if err != nil {
		logrus.Warnf("error retrieving service resources, will not port forward: %v", err)
	}

	p.portForwardResources(ctx, serviceResources)
	return nil
}

// We will port forward everything from here
// We want to wait on the pod to be created and then port forward
func (p *PortForwarder) portForwardResources(ctx context.Context, resources []latest.PortForwardResource) {
	for _, r := range resources {
		r := r
		go func() {
			if err := p.portForwardResource(ctx, r); err != nil {
				logrus.Warnf("Unable to port forward %s/%s: %v", r.Type, r.Name, err)
			}
		}()
	}
}

func (p *PortForwarder) portForwardResource(ctx context.Context, resource latest.PortForwardResource) error {
	// Get port forward entry for this resource
	entry := p.getCurrentEntry(resource)
	// Forward the resource
	return p.forward(ctx, entry)
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

// forward the portForwardEntry
func (p *PortForwarder) forward(ctx context.Context, entry *portForwardEntry) error {
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
