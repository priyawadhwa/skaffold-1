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
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type kubectlForwarder struct{}

// Forward port-forwards a portForwardEntry using kubectl port-forward
// It returns an error only if the process fails or was terminated by a signal other than SIGTERM
func (*kubectlForwarder) Forward(parentCtx context.Context, pfe *portForwardEntry) error {
	logrus.Debugf("Port forwarding %s", pfe)

	ctx, cancel := context.WithCancel(parentCtx)
	pfe.cancel = cancel

	cmd := exec.CommandContext(ctx, "kubectl", "port-forward", fmt.Sprintf("%s/%s", pfe.resource, pfe.resourceName), fmt.Sprintf("%d:%d", pfe.localPort, pfe.port), "--namespace", pfe.namespace)
	buf := &bytes.Buffer{}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		if errors.Cause(err) == context.Canceled {
			return nil
		}
		return errors.Wrapf(err, "port forwarding: %s/%s, port: %d to local port: %d, err: %s", pfe.resource, pfe.resourceName, pfe.port, pfe.localPort, buf.String())
	}
	event.PortForwarded(pfe.localPort, pfe.port, pfe.resource, pfe.resourceName, pfe.namespace, pfe.portName)

	return cmd.Wait()
}

// Terminate terminates an existing kubectl port-forward command using SIGTERM
func (*kubectlForwarder) Terminate(p *portForwardEntry) {
	logrus.Debugf("Terminating port-forward %s", p)

	if p.cancel != nil {
		p.cancel()
	}
}
