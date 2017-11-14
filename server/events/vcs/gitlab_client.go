package vcs

import (
	"fmt"
	"net/url"

	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/xanzy/go-gitlab"
)

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
func (g *GitlabClient) UpdateStatus(repo models.Repo, pull models.PullRequest, state CommitStatus, description string) error {
	const statusContext = "Atlantis"

	gitlabState := gitlab.Failed
	switch state {
	case Pending:
		gitlabState = gitlab.Pending
	case Failed:
		gitlabState = gitlab.Failed
	case Success:
		gitlabState = gitlab.Success
	}
	_, _, err := g.Client.Commits.SetCommitStatus(repo.FullName, pull.HeadCommit, &gitlab.SetCommitStatusOptions{
		State:       gitlabState,
		Context:     gitlab.String(statusContext),
		Description: gitlab.String(description),
	})
	return err
}

func (g *GitlabClient) GetMergeRequest(repoFullName string, pullNum int) (*gitlab.MergeRequest, error) {
	mr, _, err := g.Client.MergeRequests.GetMergeRequest(repoFullName, pullNum)
	return mr, err
}
