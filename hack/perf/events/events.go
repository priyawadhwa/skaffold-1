package events

import (
	"encoding/json"
	"io/ioutil"
	"path"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// Get returns a list of entries
func Get() ([]proto.LogEntry, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, errors.Wrap(err, "getting home dir")
	}
	fp := path.Join(home, constants.DefaultSkaffoldDir, constants.DefaultEventsFile)
	contents, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", fp)
	}
	var logEntries []proto.LogEntry

	if err := json.Unmarshal(contents, &logEntries); err != nil {
		return nil, errors.Wrap(err, "unmarshalling")
	}
	return logEntries, nil
}
