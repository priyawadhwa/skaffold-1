package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
	"os"
)

func main() {
	if err := getTriggers(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Get a build trigger and examine it
func getTriggers() error {
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, cloudbuild.CloudPlatformScope)
	if err != nil {
		return errors.Wrap(err, "getting google client")
	}
	cbclient, err := cloudbuild.New(client)
	if err != nil {
		return errors.Wrap(err, "getting cb client")
	}
	listCall := cbclient.Projects.Triggers.List("priya-wadhwa")
	bts, err := listCall.Do()
	if err != nil {
		return errors.Wrap(err, "getting list")
	}
	trigger := bts.Triggers[0]
	fmt.Println(trigger.TriggerTemplate.RepoName)
	fmt.Println(trigger.TriggerTemplate)
	return nil
}

func execute() error {
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, cloudbuild.CloudPlatformScope)
	if err != nil {
		return errors.Wrap(err, "getting google client")
	}
	cbclient, err := cloudbuild.New(client)
	if err != nil {
		return errors.Wrap(err, "getting service")
	}
	source := &cloudbuild.RepoSource{
		BranchName: "master",
		RepoName:   "github-priyawadhwa-trigger",
	}
	buildTrigger := &cloudbuild.BuildTrigger{
		Filename:        "cloudbuild.yaml",
		TriggerTemplate: source,
	}
	call := cbclient.Projects.Triggers.Create("priya-wadhwa", buildTrigger)
	bt, err := call.Context(ctx).Do()
	if err != nil {
		return errors.Wrap(err, "creating build trigger")
	}
	fmt.Println(bt.TriggerTemplate)
	return nil
}
