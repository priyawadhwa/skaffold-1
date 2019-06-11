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
	"io"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

// Forwarder is an interface that can modify and manage port-forward processes
type Forwarder interface {
	Start(ctx context.Context) error
	Stop()
}

type forwarders []Forwarder

func (f forwarders) Start(ctx context.Context) error {
	for _, i := range f {
		if err := i.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (f forwarders) Stop() error {
	for _, i := range f {
		i.Stop()
	}
	return nil
}

// BaseForwarder is the base port forwarder for automatic port forwarding
// and for port forwarding generic resources
type BaseForwarder struct {
	*kubectlForwarder
	output     io.Writer
	namespaces []string

	// forwardedPorts serves as a synchronized set of ports we've forwarded.
	forwardedPorts *sync.Map
}

// GetForwarders returns a list of forwarders
func GetForwarders(out io.Writer, podSelector kubernetes.PodSelector, namespaces []string, label string, automaticPodForwarding bool) forwarders {
	baseForwarder := BaseForwarder{
		kubectlForwarder: &kubectlForwarder{},
		output:           out,
		namespaces:       namespaces,
		forwardedPorts:   &sync.Map{},
	}

	var f forwarders
	pf := NewPortForwarder(baseForwarder, label)
	f = append(f, pf)

	if automaticPodForwarding {
		apf := NewAutomaticPodForwarder(baseForwarder, podSelector)
		f = append(f, apf)
	}
	return f
}
