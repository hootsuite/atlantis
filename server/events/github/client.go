// Package github provides convenience wrappers around the go-github package.
package github

// todo: rename package to vcs

import (
	"context"

	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/vcs"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_vcs_client.go VCSClientRouting

type VCSClientRouting interface {
	GetModifiedFiles(repo models.Repo, pull models.PullRequest, host vcs.Host) ([]string, error)
	CreateComment(repo models.Repo, pull models.PullRequest, comment string, host vcs.Host) error
	PullIsApproved(repo models.Repo, pull models.PullRequest, host vcs.Host) (bool, error)
	UpdateStatus(repo models.Repo, pull models.PullRequest, state vcs.CommitStatus, description string, host vcs.Host) error
}

type VCSClientRouter struct {
	GithubClient *GithubClient
	GitlabClient *GitlabClient
}

var invalidVCSErr = errors.New("Invalid VCS Host. This is a bug!")

func (v *VCSClientRouter) GetModifiedFiles(repo models.Repo, pull models.PullRequest, host vcs.Host) ([]string, error) {
	switch host {
	case vcs.Github:
		return v.GithubClient.GetModifiedFiles(repo, pull)
	case vcs.Gitlab:
		return v.GitlabClient.GetModifiedFiles(repo, pull)
	}
	return nil, invalidVCSErr
}

func (v *VCSClientRouter) CreateComment(repo models.Repo, pull models.PullRequest, comment string, host vcs.Host) error {
	switch host {
	case vcs.Github:
		return v.GithubClient.CreateComment(repo, pull, comment)
	case vcs.Gitlab:
		return v.GitlabClient.CreateComment(repo, pull, comment)
	}
	return invalidVCSErr
}

func (v *VCSClientRouter) PullIsApproved(repo models.Repo, pull models.PullRequest, host vcs.Host) (bool, error) {
	switch host {
	case vcs.Github:
		return v.GithubClient.PullIsApproved(repo, pull)
	case vcs.Gitlab:
		return v.GitlabClient.PullIsApproved(repo, pull)
	}
	return false, invalidVCSErr
}

func (v *VCSClientRouter) UpdateStatus(repo models.Repo, pull models.PullRequest, state vcs.CommitStatus, description string, host vcs.Host) error {
	switch host {
	case vcs.Github:
		return v.GithubClient.UpdateStatus(repo, pull, state, description)
	case vcs.Gitlab:
		return v.GitlabClient.UpdateStatus(repo, pull, state, description)
	}
	return invalidVCSErr
}

// GithubClient is used to perform GitHub actions.
type GithubClient struct {
	client *github.Client
	ctx    context.Context
}

// NewClient returns a valid GitHub client.
func NewClient(hostname string, user string, pass string) (*GithubClient, error) {
	tp := github.BasicAuthTransport{
		Username: strings.TrimSpace(user),
		Password: strings.TrimSpace(pass),
	}
	client := github.NewClient(tp.Client())
	// If we're using github.com then we don't need to do any additional configuration
	// for the client. It we're using Github Enterprise, then we need to manually
	// set the base url for the API.
	if hostname != "github.com" {
		baseURL := fmt.Sprintf("https://%s/api/v3/", hostname)
		base, err := url.Parse(baseURL)
		if err != nil {
			return nil, errors.Wrapf(err, "Invalid github hostname trying to parse %s", baseURL)
		}
		client.BaseURL = base
	}

	return &GithubClient{
		client: client,
		ctx:    context.Background(),
	}, nil
}

// GetModifiedFiles returns the names of files that were modified in the pull request.
// The names include the path to the file from the repo root, ex. parent/child/file.txt.
func (g *GithubClient) GetModifiedFiles(repo models.Repo, pull models.PullRequest) ([]string, error) {
	var files []string
	nextPage := 0
	for {
		opts := github.ListOptions{
			PerPage: 300,
		}
		if nextPage != 0 {
			opts.Page = nextPage
		}
		pageFiles, resp, err := g.client.PullRequests.ListFiles(g.ctx, repo.Owner, repo.Name, pull.Num, &opts)
		if err != nil {
			return files, err
		}
		for _, f := range pageFiles {
			files = append(files, f.GetFilename())
		}
		if resp.NextPage == 0 {
			break
		}
		nextPage = resp.NextPage
	}
	return files, nil
}

// CreateComment creates a comment on the pull request.
func (g *GithubClient) CreateComment(repo models.Repo, pull models.PullRequest, comment string) error {
	_, _, err := g.client.Issues.CreateComment(g.ctx, repo.Owner, repo.Name, pull.Num, &github.IssueComment{Body: &comment})
	return err
}

// PullIsApproved returns true if the pull request was approved.
func (g *GithubClient) PullIsApproved(repo models.Repo, pull models.PullRequest) (bool, error) {
	reviews, _, err := g.client.PullRequests.ListReviews(g.ctx, repo.Owner, repo.Name, pull.Num, nil)
	if err != nil {
		return false, errors.Wrap(err, "getting reviews")
	}
	for _, review := range reviews {
		if review != nil && review.GetState() == "APPROVED" {
			return true, nil
		}
	}
	return false, nil
}

// GetPullRequest returns the pull request.
func (g *GithubClient) GetPullRequest(repo models.Repo, num int) (*github.PullRequest, *github.Response, error) {
	return g.client.PullRequests.Get(g.ctx, repo.Owner, repo.Name, num)
}

// UpdateStatus updates the status badge on the pull request.
// See https://github.com/blog/1227-commit-status-api.
func (g *GithubClient) UpdateStatus(repo models.Repo, pull models.PullRequest, state vcs.CommitStatus, description string) error {
	const statusContext = "Atlantis"
	ghState := "error"
	switch state {
	case vcs.Pending:
		ghState = "pending"
	case vcs.Success:
		ghState = "success"
	case vcs.Failed:
		ghState = "failure"
	}
	status := &github.RepoStatus{
		State:       github.String(ghState),
		Description: github.String(description),
		Context:     github.String(statusContext)}
	_, _, err := g.client.Repositories.CreateStatus(g.ctx, repo.Owner, repo.Name, pull.HeadCommit, status)
	return err
}

type GitlabClient struct {
	Client *gitlab.Client
}

// GetModifiedFiles returns the names of files that were modified in the merge request.
// The names include the path to the file from the repo root, ex. parent/child/file.txt.
func (g *GitlabClient) GetModifiedFiles(repo models.Repo, pull models.PullRequest) ([]string, error) {
	const maxPerPage = 100
	var files []string
	nextPage := 1
	// Constructing the api url by hand so we can do pagination.
	apiURL := fmt.Sprintf("projects/%s/merge_requests/%d/changes", url.QueryEscape(repo.FullName), pull.Num)
	for {
		opts := gitlab.ListOptions{
			Page:    nextPage,
			PerPage: maxPerPage,
		}
		req, err := g.Client.NewRequest("GET", apiURL, opts, nil)
		if err != nil {
			return nil, err
		}
		mr := new(gitlab.MergeRequest)
		resp, err := g.Client.Do(req, mr)
		if err != nil {
			return nil, err
		}

		for _, f := range mr.Changes {
			files = append(files, f.NewPath)
		}
		if resp.NextPage == 0 {
			break
		}
		nextPage = resp.NextPage
	}

	return files, nil
}

// CreateComment creates a comment on the merge request.
func (g *GitlabClient) CreateComment(repo models.Repo, pull models.PullRequest, comment string) error {
	_, _, err := g.Client.Notes.CreateMergeRequestNote(repo.FullName, pull.Num, &gitlab.CreateMergeRequestNoteOptions{Body: gitlab.String(comment)})
	return err
}

// PullIsApproved returns true if the merge request was approved.
func (g *GitlabClient) PullIsApproved(repo models.Repo, pull models.PullRequest) (bool, error) {
	approvals, _, err := g.Client.MergeRequests.GetMergeRequestApprovals(repo.FullName, pull.Num)
	if err != nil {
		return false, err
	}
	if approvals.ApprovalsMissing > 0 {
		return false, nil
	}
	return true, nil
}

// UpdateStatus updates the build status of a commit.
func (g *GitlabClient) UpdateStatus(repo models.Repo, pull models.PullRequest, state vcs.CommitStatus, description string) error {
	const statusContext = "Atlantis"

	gitlabState := gitlab.Failed
	switch state {
	case vcs.Pending:
		gitlabState = gitlab.Pending
	case vcs.Failed:
		gitlabState = gitlab.Failed
	case vcs.Success:
		gitlabState = gitlab.Success
	}
	_, _, err := g.Client.Commits.SetCommitStatus(repo.FullName, pull.HeadCommit, &gitlab.SetCommitStatusOptions{
		State:       gitlabState,
		Context:     gitlab.String(statusContext),
		Description: gitlab.String(description),
	})
	return err
}
