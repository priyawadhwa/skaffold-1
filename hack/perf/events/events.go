package events

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var (
	eventsFile string
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

func Cleanup() {
	defer os.Remove(eventsFile)
	eventsFile = ""
}

func File() (string, error) {
	if eventsFile != "" {
		return eventsFile, nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("homedir: %w", err)
	}
	f, err := ioutil.TempFile(home, "events")
	if err != nil {
		return "", fmt.Errorf("temp file: %w", err)
	}
	defer f.Close()
	eventsFile = f.Name()
	return f.Name(), nil
}
