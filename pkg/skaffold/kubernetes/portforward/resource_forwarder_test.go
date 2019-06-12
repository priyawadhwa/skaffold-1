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
	"sync"
)

type testForwarder struct {
	forwardedEntries map[string]*portForwardEntry
	forwardedPorts   map[int32]bool

	forwardErr error
}

func (f *testForwarder) Forward(ctx context.Context, pfe *portForwardEntry) error {
	f.forwardedEntries[pfe.key()] = pfe
	f.forwardedPorts[pfe.localPort] = true
	return f.forwardErr
}

func (f *testForwarder) Terminate(pfe *portForwardEntry) {
	delete(f.forwardedEntries, pfe.key())
	delete(f.forwardedPorts, pfe.resource.Port)
}

func newTestForwarder(forwardErr error) *testForwarder {
	return &testForwarder{
		forwardedEntries: map[string]*portForwardEntry{},
		forwardedPorts:   map[int32]bool{},
		forwardErr:       forwardErr,
	}
}

func mockRetrieveAvailablePort(taken map[int]struct{}, availablePorts []int) func(int, *sync.Map) int {
	// Return first available port in ports that isn't taken
	return func(int, *sync.Map) int {
		for _, p := range availablePorts {
			if _, ok := taken[p]; ok {
				continue
			}
			taken[p] = struct{}{}
			return p
		}
		return -1
	}
}
