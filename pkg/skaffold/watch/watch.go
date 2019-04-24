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

package watch

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Factory creates Watcher instances.
type Factory func() Watcher

// Watcher monitors files changes for multiples components.
type Watcher interface {
	Register(deps func() ([]string, error), onChange func(Events)) error
	Run(ctx context.Context, out io.Writer, onChange func() error) error
	Subscribe(s Subscriber)
}

type watchList struct {
	components  []*component
	trigger     Trigger
	subscribers []Subscriber
}

// NewWatcher creates a new Watcher.
func NewWatcher(trigger Trigger) Watcher {
	return &watchList{
		trigger: trigger,
	}
}

type component struct {
	deps     func() ([]string, error)
	onChange func(Events)
	state    FileMap
	events   Events
}

type Subscriber interface {
	HandleEvent(event string, data interface{})
}

func (w *watchList) Subscribe(s Subscriber) {
	w.subscribers = append(w.subscribers, s)
}

func (w *watchList) HandleEvent(event string, data interface{}) {
	for _, s := range w.subscribers {
		s.HandleEvent(event, data)
	}
}

// Register adds a new component to the watch list.
func (w *watchList) Register(deps func() ([]string, error), onChange func(Events)) error {
	state, err := Stat(deps)
	if err != nil {
		return errors.Wrap(err, "listing files")
	}

	w.components = append(w.components, &component{
		deps:     deps,
		onChange: onChange,
		state:    state,
	})
	return nil
}

// Run watches files until the context is cancelled or an error occurs.
func (w *watchList) Run(ctx context.Context, out io.Writer, onChange func() error) error {
	ctxTrigger, cancelTrigger := context.WithCancel(ctx)
	defer cancelTrigger()

	t, err := w.trigger.Start(ctxTrigger)
	if err != nil {
		return errors.Wrap(err, "unable to start trigger")
	}

	changedComponents := map[int]bool{}

	w.trigger.WatchForChanges(out)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t:
			changed := 0
			for i, component := range w.components {
				state, err := Stat(component.deps)
				if err != nil {
					return errors.Wrap(err, "listing files")
				}
				e := events(component.state, state)

				if e.HasChanged() {
					changedComponents[i] = true
					component.state = state
					component.events = e
					changed++
				}
			}

			// Rapid file changes that are more frequent than the poll interval would trigger
			// multiple rebuilds.
			// To prevent that, we debounce changes that happen too quickly
			// by waiting for a full turn where nothing happens and trigger a rebuild for
			// the accumulated changes.
			debounce := w.trigger.Debounce()
			if (!debounce && changed > 0) || (debounce && changed == 0 && len(changedComponents) > 0) {
				var deps []string
				for i, component := range w.components {
					if changedComponents[i] {
						for d := range component.state {
							deps = append(deps, d)
						}
						component.onChange(component.events)
					}
				}
				fmt.Println("sending over", deps)
				w.HandleEvent("change", deps)

				if err := onChange(); err != nil {
					return errors.Wrap(err, "calling final callback")
				}

				changedComponents = map[int]bool{}
				w.trigger.WatchForChanges(out)
			}
		}
	}
}
