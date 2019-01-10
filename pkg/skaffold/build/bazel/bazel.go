package bazel

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// BazelArtifact describes an artifact built with Bazel.
type BazelArtifact struct {
	BuildTarget string   `yaml:"target,omitempty"`
	BuildArgs   []string `yaml:"args,omitempty"`
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

// Labels are labels specific to Bazel.
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "bazel",
	}
}

func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact, env latest.ExecutionEnvironment) ([]build.Artifact, error) {

	if env.Name == "local" {
		return build.InSequence(ctx, out, tagger, artifacts, b.buildArtifact)
	}
	return nil, errors.Errorf("%s is not a supported environment for builder %s", env.Name, "bazel")
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, tagger tag.Tagger, artifact *latest.Artifact) (string, error) {
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

	data, err := yaml.Marshal(artifact.Plugin.Properties)
	if err != nil {
		return "", errors.Wrap(err, "marshalling properties")
	}
	var properties *BazelArtifact
	if err := yaml.Unmarshal(data, &properties); err != nil {
		return "", err
	}

	args := []string{"build"}
	args = append(args, properties.BuildArgs...)
	args = append(args, properties.BuildTarget)
	workspace := artifact.Workspace

	fmt.Println("workspace is", workspace)
	fmt.Println(os.Getwd())

	cmd := exec.CommandContext(ctx, "bazel", args...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "running command")
	}

	bazelBin, err := bazelBin(ctx, workspace)
	if err != nil {
		return "", errors.Wrap(err, "getting path of bazel-bin")
	}

	tarPath := buildTarPath(properties.BuildTarget)
	imageTar, err := os.Open(filepath.Join(bazelBin, tarPath))
	if err != nil {
		return "", errors.Wrap(err, "opening image tarball")
	}
	defer imageTar.Close()

	ref := buildImageTag(properties.BuildTarget)

	imageID, err := b.Builder.LocalDocker.Load(ctx, out, imageTar, ref)
	if err != nil {
		return "", errors.Wrap(err, "loading image into docker daemon")
	}

	return imageID, nil
}

func bazelBin(ctx context.Context, workspace string) (string, error) {
	cmd := exec.CommandContext(ctx, "bazel", "info", "bazel-bin")
	cmd.Dir = workspace

	buf, err := util.RunCmdOut(cmd)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(buf)), nil
}

func trimTarget(buildTarget string) string {
	//TODO(r2d4): strip off leading //:, bad
	trimmedTarget := strings.TrimPrefix(buildTarget, "//")
	// Useful if root target "//:target"
	trimmedTarget = strings.TrimPrefix(trimmedTarget, ":")

	return trimmedTarget
}

func buildTarPath(buildTarget string) string {
	tarPath := trimTarget(buildTarget)
	tarPath = strings.Replace(tarPath, ":", string(os.PathSeparator), 1)

	return tarPath
}

func buildImageTag(buildTarget string) string {
	imageTag := trimTarget(buildTarget)
	imageTag = strings.TrimPrefix(imageTag, ":")

	//TODO(r2d4): strip off trailing .tar, even worse
	imageTag = strings.TrimSuffix(imageTag, ".tar")

	if strings.Contains(imageTag, ":") {
		return fmt.Sprintf("bazel/%s", imageTag)
	}

	return fmt.Sprintf("bazel:%s", imageTag)
}
