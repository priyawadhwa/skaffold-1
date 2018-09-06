package main

import (
	"fmt"
	"github.com/pkg/errors"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
	"net/http"
	"os"
)

func main() {
	if err := execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func execute() error {
	tr := &http.Client{}
	client, err := cloudbuild.New(tr)
	if err != nil {
		return errors.Wrap(err, "getting service")
	}
	triggerservice := cloudbuild.NewProjectsTriggersService(client)
	source := &cloudbuild.Source{
		RepoSource: &cloudbuild.RepoSource{
			BranchName: "master",
			RepoName:   "priyawadhwa/trigger",
		},
	}
	build := &cloudbuild.Build{
		Source: source,
	}
	buildTrigger := &cloudbuild.BuildTrigger{
		Build: build,
	}
	call := triggerservice.Create("priya-wadhwa", buildTrigger)
	bt, err := call.Do()
	if err != nil {
		return errors.Wrap(err, "creating build trigger")
	}
	fmt.Println("succesfully created", bt.Id)
	return nil
}
