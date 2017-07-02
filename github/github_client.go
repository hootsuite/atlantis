package github

import (
	"context"

	"github.com/google/go-github/github"
	"github.com/hootsuite/atlantis/models"
	"github.com/pkg/errors"
	"strings"
	"fmt"
	"net/url"
)

type Client struct {
	client *github.Client
	ctx    context.Context
}

func NewClient(hostname string, user string, pass string) (*Client, error) {
	tp := github.BasicAuthTransport{
		Username: strings.TrimSpace(user),
		Password: strings.TrimSpace(pass),
	}
	client := github.NewClient(tp.Client())
	// If we're using github.com then we don't need to do any additional configuration
	// for the client. It we're using Github Enterprise, then we need to manually
	// set the base url for the API
	if hostname != "github.com" {
		baseURL := fmt.Sprintf("https://%s/api/v3/", hostname)
		base, err := url.Parse(baseURL)
		if err != nil {
			return nil, errors.Wrapf(err, "Invalid github hostname trying to parse %s", baseURL)
		}
		client.BaseURL = base
	}

	return &Client{
		client: client,
		ctx:    context.Background(),
	}, nil
}

// GetModifiedFiles returns the names of files that were modified in the pull request.
// The names include the path to the file from the repo root, ex. parent/child/file.txt
func (c *Client) GetModifiedFiles(repo models.Repo, pull models.PullRequest) ([]string, error) {
	var files []string
	nextPage := 0
	for {
		opts := github.ListOptions{
			PerPage: 300,
		}
		if nextPage != 0 {
			opts.Page = nextPage
		}
		pageFiles, resp, err := c.client.PullRequests.ListFiles(c.ctx, repo.Owner, repo.Name, pull.Num, &opts)
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

func (c *Client) CreateComment(repo models.Repo, pull models.PullRequest, comment string) error {
	_, _, err := c.client.Issues.CreateComment(c.ctx, repo.Owner, repo.Name, pull.Num, &github.IssueComment{Body: &comment})
	return err
}

func (c *Client) PullIsApproved(repo models.Repo, pull models.PullRequest) (bool, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(c.ctx, repo.Owner, repo.Name, pull.Num, nil)
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

func (c *Client) GetPullRequest(repo models.Repo, num int) (*github.PullRequest, *github.Response, error) {
	return c.client.PullRequests.Get(c.ctx, repo.Owner, repo.Name, num)
}

func (c *Client) UpdateStatus(repo models.Repo, pull models.PullRequest, state string, description string, context string) error {
	status := &github.RepoStatus{
		State:       github.String(state),
		Description: github.String(description),
		Context:     github.String(context)}
	_, _, err := c.client.Repositories.CreateStatus(c.ctx, repo.Owner, repo.Name, pull.HeadCommit, status)
	return err
}
