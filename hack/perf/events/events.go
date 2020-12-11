package events

import (
	"bytes"
	"fmt"
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
func Get(contents []byte) ([]proto.LogEntry, error) {
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

func GetFromFile(fp string) ([]proto.LogEntry, error) {
	contents, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", fp)
	}
	return Get(contents)
}

func File() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("homedir: %w", err)
	}
	return path.Join(home, constants.DefaultSkaffoldDir, constants.DefaultEventsFile), nil
}
