package skaffold

import (
	"fmt"
	"net/http"
	"os/exec"
	"path"

	"github.com/GoogleContainerTools/skaffold/hack/perf/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

var ()

func Dev(app config.Application) error {
	logrus.Infof("Starting skaffold dev on %s...", app.Name)
	eventsFile, err := EventsFile()
	if err != nil {
		return fmt.Errorf("events file: %w", err)
	}
	port := util.GetAvailablePort(util.Loopback, 8080, nil)
	cmd := exec.Command("skaffold", "dev", "--enable-rpc", fmt.Sprintf("--rpc-port=%v", port), fmt.Sprintf("--event-log-file=%s", eventsFile))
	logrus.Infof("Running [%v]", cmd.Args)
	go cmd.Start()
	return nil
}

func waitForDevLoopComplete(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s", util.Loopback, port))
	if err != nil {
	}
	return false
}

func EventsFile() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("homedir: %w", err)
	}
	return path.Join(home, constants.DefaultSkaffoldDir, constants.DefaultEventsFile), nil
}
