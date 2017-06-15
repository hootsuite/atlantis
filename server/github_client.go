package server

import (
	"fmt"
	"github.com/google/go-github/github"
	"context"
)

type GithubClient struct {
	client *github.Client
	ctx    context.Context
}

const (
	statusContext = "Atlantis"
	PendingStatus = "pending"
	SuccessStatus = "success"
	ErrorStatus = "error"
	FailureStatus = "failure"
)

func (g *GithubClient) UpdateStatus(repo Repo, pull PullRequest, status string, description string) {
	repoStatus := github.RepoStatus{State: github.String(status), Description: github.String(description), Context: github.String(statusContext)}
	g.client.Repositories.CreateStatus(g.ctx, repo.Owner, repo.Name, pull.HeadCommit, &repoStatus)
	// todo: deal with error updating status
}

func (g *GithubClient) GetModifiedFiles(repo Repo, pull PullRequest) ([]string, error) {
	var files = []string{}
	comparison, _, err := g.client.Repositories.CompareCommits(g.ctx, repo.Owner, repo.Name, pull.BaseCommit, pull.HeadCommit)
	if err != nil {
		return files, err
	}
	for _, file := range comparison.Files {
		files = append(files, *file.Filename)
	}
	return files, nil
}

func (g *GithubClient) CreateComment(ctx *CommandContext, comment string) error {
	_, _, err := g.client.Issues.CreateComment(g.ctx, ctx.Repo.Owner, ctx.Repo.Name, ctx.Pull.Num, &github.IssueComment{Body: &comment})
	return err
}

func (g *GithubClient) PullIsApproved(repo Repo, pull PullRequest) (bool, error) {
	// todo: move back to using g.client.PullRequests.ListReviews when we update our GitHub enterprise version
	// to where we don't need to include the custom accept header
	u := fmt.Sprintf("repos/%v/%v/pulls/%d/reviews", repo.Owner, repo.Name, pull.Num)
	req, err := g.client.NewRequest("GET", u, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github.black-cat-preview+json")

	var reviews []*github.PullRequestReview
	_, err = g.client.Do(g.ctx, req, &reviews)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve reviews: %v", err)
	}
	for _, review := range reviews {
		if review != nil && review.State != nil && *review.State == "APPROVED" {
			return true, nil
		}
	}
	return false, nil
}

func (g *GithubClient) GetPullRequest(repo Repo, num int) (*github.PullRequest, *github.Response, error) {
	return g.client.PullRequests.Get(g.ctx, repo.Owner, repo.Name, num)
}
