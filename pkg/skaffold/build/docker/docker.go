package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Builder builds artifacts with Google Cloud Build.
type Builder struct {
	Builder *local.Builder
}

// DockerArtifact describes an artifact built from a Dockerfile,
// usually using `docker build`.
type DockerArtifact struct {
	DockerfilePath string             `yaml:"dockerfile,omitempty"`
	BuildArgs      map[string]*string `yaml:"buildArgs,omitempty"`
	CacheFrom      []string           `yaml:"cacheFrom,omitempty"`
	Target         string             `yaml:"target,omitempty"`
}

// NewBuilder creates a new Builder that builds artifacts with Google Cloud Build.
func NewBuilder() *Builder {
	builder, err := local.NewBuilder(&latest.LocalBuild{}, "")
	if err != nil {
		panic(err)
	}
	return &Builder{
		Builder: builder,
	}
}

// Labels are labels specific to Docker.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "docker",
	}
}

func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact, env latest.ExecutionEnvironment) ([]build.Artifact, error) {
	if env.Name == "local" {
		return build.InSequence(ctx, out, tagger, artifacts, b.buildArtifactLocal)
	}
	return nil, errors.Errorf("%s is not a supported environment for builder %s", env.Name, "docker")
}

func (b *Builder) buildArtifactLocal(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
	digest, err := b.buildLocal(ctx, out, tagger, artifact)
	if err != nil {
		return "", errors.Wrap(err, "build artifact")
	}

	if b.Builder.AlreadyTagged == nil {
		b.Builder.AlreadyTagged = make(map[string]string)
	}
	if tag, present := b.Builder.AlreadyTagged[digest]; present {
		return tag, nil
	}

	tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, tag.Options{
		ImageName: artifact.ImageName,
		Digest:    digest,
	})
	if err != nil {
		return "", errors.Wrap(err, "generating tag")
	}

	if err := b.retagAndPush(ctx, out, digest, tag, artifact); err != nil {
		return "", errors.Wrap(err, "tagging")
	}

	b.Builder.AlreadyTagged[digest] = tag

	return tag, nil
}

func (b *Builder) retagAndPush(ctx context.Context, out io.Writer, initialTag string, newTag string, artifact *latest.Artifact) error {
	if b.Builder.PushImages && (artifact.JibMavenArtifact != nil || artifact.JibGradleArtifact != nil) {
		if err := docker.AddTag(initialTag, newTag); err != nil {
			return errors.Wrap(err, "tagging image")
		}
		return nil
	}

	if err := b.Builder.LocalDocker.Tag(ctx, initialTag, newTag); err != nil {
		return err
	}

	if b.Builder.PushImages {
		if _, err := b.Builder.LocalDocker.Push(ctx, out, newTag); err != nil {
			return errors.Wrap(err, "pushing")
		}
	}

	return nil
}

func (b *Builder) buildLocal(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {

	var properties *DockerArtifact
	if err := yaml.Unmarshal(artifact.Plugin.Contents, &properties); err != nil {
		return "", err
	}

	initialTag := util.RandomID()
	workspace := artifact.Workspace

	if b.Builder.Cfg.UseDockerCLI || b.Builder.Cfg.UseBuildkit {

		dockerfilePath, err := docker.NormalizeDockerfilePath(workspace, properties.DockerfilePath)
		if err != nil {
			return "", errors.Wrap(err, "normalizing dockerfile path")
		}

		args := []string{"build", workspace, "--file", dockerfilePath, "-t", initialTag}
		args = append(args, GetBuildArgs(properties)...)

		fmt.Printf("Args are %v \n", args)

		cmd := exec.CommandContext(ctx, "docker", args...)
		if b.Builder.Cfg.UseBuildkit {
			cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
		}
		cmd.Stdout = out
		cmd.Stderr = out

		if err := util.RunCmd(cmd); err != nil {
			return "", errors.Wrap(err, "running build")
		}

		return b.Builder.LocalDocker.ImageID(ctx, initialTag)
	}

	// unmarshal to latest.DockerArtifact to make life easier
	var da *latest.DockerArtifact
	if err := yaml.Unmarshal(artifact.Plugin.Contents, &da); err != nil {
		return "", err
	}

	return b.Builder.LocalDocker.Build(ctx, out, workspace, da, initialTag)
}

// GetBuildArgs gives the build args flags for docker build.
func GetBuildArgs(a *DockerArtifact) []string {
	var args []string

	var keys []string
	for k := range a.BuildArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		args = append(args, "--build-arg")

		v := a.BuildArgs[k]
		if v == nil {
			args = append(args, k)
		} else {
			args = append(args, fmt.Sprintf("%s=%s", k, *v))
		}
	}

	for _, from := range a.CacheFrom {
		args = append(args, "--cache-from", from)
	}

	if a.Target != "" {
		args = append(args, "--target", a.Target)
	}

	return args
}
