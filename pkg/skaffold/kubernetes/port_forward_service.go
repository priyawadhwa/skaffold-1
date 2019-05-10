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
	"context"
	"fmt"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serviceResource = "svc"
)

func (p *PortForwarder) portForwardServices(ctx context.Context) error {
	// Gather all services deployed by this app by filtering by the app.kubernetes.io/managed-by label
	client, err := GetClientset()
	if err != nil {
		return errors.Wrap(err, "gettig kubernetes clienset")
	}
	svcs, err := client.CoreV1().Services("").List(metav1.ListOptions{
		LabelSelector: p.label,
	})
	if err != nil {
		return errors.Wrapf(err, "getting svcs with label selector %s", p.label)
	}

	if svcs == nil {
		return nil
	}

	for _, svc := range svcs.Items {
		for _, port := range svc.Spec.Ports {
			go func() {
				svc := svc
				port := port
				fmt.Println("trying to forward")
				if err := p.forwardSvc(ctx, svc, port); err != nil {
					fmt.Println("error trying to forward")
					logrus.Warn("error forwarding service %s: %v", svc.Name, err)
				}
			}()

		}
	}
	return nil
}

func (p *PortForwarder) forwardSvc(ctx context.Context, svc v1.Service, port v1.ServicePort) error {
	resourceVersion, err := strconv.Atoi(svc.ResourceVersion)
	if err != nil {
		return errors.Wrap(err, "converting resource version to integer")
	}

	entry := p.getCurrentEntry(serviceResource, svc.Name, svc.Namespace, "", string(port.Port), port.TargetPort.IntVal, resourceVersion)
	if entry.port != entry.localPort {
		color.Yellow.Fprintf(p.output, "Forwarding service %s to local port %d.\n", svc.Name, entry.localPort)
	}
	if err := p.forward(ctx, entry); err != nil {
		return errors.Wrap(err, "failed to forward port")
	}
	return nil
}
