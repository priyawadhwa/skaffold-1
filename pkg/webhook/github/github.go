/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"log"
	"os"
)

// GithubClient provides the context and client with necessary auth
// for interacting with the Github API
type GithubClient struct {
	ctx context.Context
	*github.Client
}

// NewGithubClient returns a github client with the necessary auth
func NewGithubClient() *GithubClient {
	githubToken := os.Getenv(constants.GithubAccessToken)
	// Setup the token for github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	// Return a client instance from github
	client := github.NewClient(tc)
	return &GithubClient{
		Client: client,
		ctx:    context.Background(),
	}
}

// CommentOnPR comments message on the PR
func (g *GithubClient) CommentOnPR(pr *github.PullRequestEvent, message string) error {
	comment := &github.IssueComment{
		Body: &[]string{message}[0],
	}

	log.Printf("Creating comment on PR %d", pr.PullRequest.GetNumber())
	_, _, err := g.Client.Issues.CreateComment(g.ctx, constants.GithubOwner, constants.GithubRepo, pr.PullRequest.GetNumber(), comment)
	if err != nil {
		return errors.Wrap(err, "creating github comment")
	}
	log.Printf("Succesfully commented on PR.")
	return nil
}

// RemoveLabelFromPR removes label from pr
func (g *GithubClient) RemoveLabelFromPR(pr *github.PullRequestEvent, label string) error {
	_, err := g.Client.Issues.DeleteLabel(g.ctx, constants.GithubOwner, constants.GithubRepo, label)
	if err != nil {
		return errors.Wrap(err, "deleting label")
	}
	log.Printf("Successfully deleted label from PR %d", pr.GetNumber())
	return nil
}
