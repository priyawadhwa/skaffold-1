package shared

import (
	"context"
	"fmt"
	"io"
	"net/rpc"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
)

// Here is an implementation that talks over RPC
type BuilderRPC struct {
	client *rpc.Client
}

func (b *BuilderRPC) Labels() map[string]string {
	var resp map[string]string
	err := b.client.Call("Plugin.Labels", new(interface{}), &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}
	return resp
}

func (b *BuilderRPC) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var resp []build.Artifact
	fmt.Println("registered")
	args := BuilderArgs{
		Tagger:    tagger.String(),
		Artifacts: artifacts,
	}
	err := b.client.Call("Plugin.Build", args, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Here is the RPC server that BuilderRPC talks to, conforming to
// the requirements of net/r
type BuilderRPCServer struct {
	Impl build.Builder
}

func (s *BuilderRPCServer) Labels(args interface{}, resp *map[string]string) error {
	fmt.Println("calling labels from server")
	*resp = s.Impl.Labels()
	return nil
}

func (s *BuilderRPCServer) Build(b BuilderArgs, resp *[]build.Artifact) error {
	fmt.Println("calling build from server")

	tagger := tag.RetrieveTagger(b.Tagger)
	artifacts, err := s.Impl.Build(context.Background(), os.Stderr, tagger, b.Artifacts)
	if err != nil {
		return errors.Wrap(err, "building artifacts")
	}
	*resp = artifacts
	return nil
}

type BuilderArgs struct {
	Tagger    string
	Artifacts []*latest.Artifact
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a GreeterRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return GreeterRPC for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type BuilderPlugin struct {
	// Impl Injection
	Impl build.Builder
}

func (p *BuilderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &BuilderRPCServer{Impl: p.Impl}, nil
}

func (BuilderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &BuilderRPC{client: c}, nil
}
