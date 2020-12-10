package skaffold

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"path"
	"time"

	"github.com/GoogleContainerTools/skaffold/hack/perf/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

func Dev(ctx context.Context, app config.Application) error {
	logrus.Infof("Starting skaffold dev on %s...", app.Name)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eventsFile, err := EventsFile()
	if err != nil {
		return fmt.Errorf("events file: %w", err)
	}
	port := util.GetAvailablePort(util.Loopback, 8080, &util.PortSet{})

	buf := bytes.NewBuffer([]byte{})
	cmd := exec.CommandContext(ctx, "skaffold", "dev", "--enable-rpc", fmt.Sprintf("--rpc-port=%v", port), fmt.Sprintf("--event-log-file=%s", eventsFile))
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
	if err := waitForDevLoopComplete(ctx, port); err != nil {
		return fmt.Errorf("waiting for dev loop complete: %w: %s", err, buf.String())
	}
	fmt.Println("Dev loop complete woot")
	return nil
}

func waitForDevLoopComplete(ctx context.Context, port int) error {
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
		if entry.GetEvent().GetDevLoopEvent().GetStatus() == event.Succeeded {
			break
		}
	}
	return nil
}

func EventsFile() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("homedir: %w", err)
	}
	return path.Join(home, constants.DefaultSkaffoldDir, constants.DefaultEventsFile), nil
}
