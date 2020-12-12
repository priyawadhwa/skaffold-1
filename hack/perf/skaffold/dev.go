package skaffold

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/hack/perf/config"
	"github.com/GoogleContainerTools/skaffold/hack/perf/events"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

func Dev(ctx context.Context, app config.Application) error {
	logrus.Infof("Starting skaffold dev on %s...", app.Name)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := copyAppToTmpDir(&app); err != nil {
		return fmt.Errorf("copying app to temp dir: %w", err)
	}
	defer os.Remove(app.Context)

	eventsFile, err := events.File()
	if err != nil {
		return fmt.Errorf("events file: %w", err)
	}
	port := util.GetAvailablePort(util.Loopback, 8080, &util.PortSet{})

	buf := bytes.NewBuffer([]byte{})
	cmd := exec.CommandContext(ctx, "skaffold", "dev", "--enable-rpc", fmt.Sprintf("--rpc-port=%v", port), fmt.Sprintf("--event-log-file=%s", eventsFile), "--cache-artifacts=false")
	cmd.Dir = app.Context
	cmd.Stdout = buf
	cmd.Stderr = buf

	logrus.Infof("Running [%v] in %v", cmd.Args, cmd.Dir)
	go func() {
		defer cancel()
		if err := cmd.Run(); err != nil {
			logrus.Infof("skaffold dev failed: %v", err)
		}
	}()
	if err := waitForDevLoopComplete(ctx, 0, port); err != nil {
		return fmt.Errorf("waiting for dev loop complete: %w: %s", err, buf.String())
	}
	logrus.Info("Dev loop iteration 1 is complete, initiating inner loop...")
	if err := kickoffDevLoop(ctx, app); err != nil {
		return fmt.Errorf("kicking off dev loop: %w", err)
	}
	if err := waitForDevLoopComplete(ctx, 1, port); err != nil {
		return fmt.Errorf("waiting for dev loop complete: %w: %s", err, buf.String())
	}
	logrus.Infof("successfully ran inner dev loop, killing skaffold...")
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("killing skaffold: %w", err)
	}

	return wait.Poll(time.Second, 2*time.Minute, func() (bool, error) {
		_, err := os.Stat(eventsFile)
		return err == nil, nil
	})
}

func copyAppToTmpDir(app *config.Application) error {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	logrus.Infof("copying %v to temp location %v", app.Context, dir)
	if err := copy.Copy(app.Context, dir); err != nil {
		return fmt.Errorf("copying dir: %w", err)
	}
	logrus.Infof("using temp directory %v as app directory", dir)
	app.Context = dir
	return nil
}

func kickoffDevLoop(ctx context.Context, app config.Application) error {
	args := strings.Split(app.Dev.Command, " ")
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = app.Context

	logrus.Infof("Running [%v] in %v", cmd.Args, cmd.Dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running %v: output %s: %w", cmd.Args, string(output), err)
	}
	return nil
}

func waitForDevLoopComplete(ctx context.Context, iteration, port int) error {
	var (
		conn   *grpc.ClientConn
		err    error
		client proto.SkaffoldServiceClient
	)

	if err := wait.Poll(time.Second, 2*time.Minute, func() (bool, error) {
		conn, err = grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
		if err != nil {
			return false, nil
		}
		client = proto.NewSkaffoldServiceClient(conn)
		return true, nil
	}); err != nil {
		return fmt.Errorf("getting grpc client connection: %w", err)
	}
	defer conn.Close()

	logrus.Infof("successfully connected to grpc client")

	// read the event log stream from the skaffold grpc server
	var stream proto.SkaffoldService_EventsClient
	for i := 0; i < 10; i++ {
		stream, err = client.Events(ctx, &empty.Empty{})
		if err != nil {
			log.Printf("error getting stream, retrying: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}
	}
	if stream == nil {
		log.Fatalf("error retrieving event log: %v\n", err)
	}

	devLoopIterations := 0
	for {
		if ctx.Err() == context.Canceled {
			return context.Canceled
		}
		entry, err := stream.Recv()
		if err != nil {
			log.Fatalf("error receiving entry from stream: %s", err)
		}
		log.Printf("received event: %v", entry)
		if entry.GetEvent().GetDevLoopEvent() == nil {
			continue
		}
		if entry.GetEvent().GetDevLoopEvent().GetStatus() != event.Succeeded {
			continue
		}
		if devLoopIterations == iteration {
			break
		}
		devLoopIterations++
	}
	return nil
}
