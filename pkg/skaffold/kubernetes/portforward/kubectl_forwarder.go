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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type kubectlForwarder struct{}

// Forward port-forwards a pod using kubectl port-forward
// It returns an error only if the process fails or was terminated by a signal other than SIGTERM
func (*kubectlForwarder) Forward(parentCtx context.Context, pfe *portForwardEntry) error {
	logrus.Debugf("Port forwarding %v", pfe)

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
	logrus.Debugf("Terminating port-forward %v", p)

	if p.cancel != nil {
		p.cancel()
	}
}
