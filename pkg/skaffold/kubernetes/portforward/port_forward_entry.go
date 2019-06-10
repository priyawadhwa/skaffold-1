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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type portForwardEntry struct {
	resourceVersion int
	resource        latest.PortForwardResource
	podName         string
	containerName   string
	portName        string
	localPort       int32

	cancel context.CancelFunc
}

// Key is an identifier for the lock on a port during the skaffold dev cycle.
func (p *portForwardEntry) key() string {
	return fmt.Sprintf("%s-%s-%s-%d", p.resource.Type, p.resource.Name, p.resource.Namespace, p.resource.Port)
}

// Key is an identifier for the lock on a port during the skaffold dev cycle.
func (p *portForwardEntry) podKey() string {
	return fmt.Sprintf("%s-%s-%s-%d", p.containerName, p.resource.Namespace, p.portName, p.resource.Port)
}

// String is a utility function that returns the port forward entry as a user-readable string
func (p *portForwardEntry) String() string {
	fmt.Println(*p)
	return fmt.Sprintf("%s-%s-%s-%d", p.resource.Type, p.resource.Name, p.resource.Namespace, p.resource.Port)
}
