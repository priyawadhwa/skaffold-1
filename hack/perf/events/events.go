package events

import (
	"bytes"
	"io/ioutil"
	"path"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/golang/protobuf/jsonpb"
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
	entries := strings.Split(string(contents), "\n")
	var logEntries []proto.LogEntry
	unmarshaller := jsonpb.Unmarshaler{}
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		var logEntry proto.LogEntry
		buf := bytes.NewBuffer([]byte(entry))
		if err := unmarshaller.Unmarshal(buf, &logEntry); err != nil {
			return nil, errors.Wrap(err, "unmarshalling")
		}
		logEntries = append(logEntries, logEntry)
	}
	return logEntries, nil
}
