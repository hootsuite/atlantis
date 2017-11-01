package events_test

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hootsuite/atlantis/server/events"
	ghMocks "github.com/hootsuite/atlantis/server/events/github/mocks"
	eventMocks "github.com/hootsuite/atlantis/server/events/mocks"
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/logging"
	. "github.com/hootsuite/atlantis/testing"
	. "github.com/petergtz/pegomock"
)

func TestExecute_RequireApprovalError(t *testing.T) {
	t.Log("If checking whether pull request is approved there is an error we are returning it")

	g := ghMocks.NewMockClient()
	ctx := &events.CommandContext{
		BaseRepo: models.Repo{},
		Pull:     models.PullRequest{},
	}
	applyExecutor := &events.ApplyExecutor{
		Github:          g,
		RequireApproval: true,
	}
	When(g.PullIsApproved(ctx.BaseRepo, ctx.Pull)).ThenReturn(false, errors.New("error"))
	response := applyExecutor.Execute(ctx)
	Equals(t, "checking if pull request was approved: error", response.Error.Error())
}

func TestExecute_RequireApprovalIfApproved(t *testing.T) {
	t.Log("If the pull request is not approved there is a failure and we are returning it")

	g := ghMocks.NewMockClient()
	ctx := &events.CommandContext{
		BaseRepo: models.Repo{},
		Pull:     models.PullRequest{},
	}
	applyExecutor := &events.ApplyExecutor{
		Github:          g,
		RequireApproval: true,
	}
	When(g.PullIsApproved(ctx.BaseRepo, ctx.Pull)).ThenReturn(false, nil)
	response := applyExecutor.Execute(ctx)
	Equals(t, "Pull request must be approved before running apply.", response.Failure)
}

func TestExecute_GetWorkspaceError(t *testing.T) {
	t.Log("If while getting workspace there is an error we should return a failure")

	w := eventMocks.NewMockWorkspace()
	ctx := &events.CommandContext{
		BaseRepo: models.Repo{},
		Pull:     models.PullRequest{},
		Command:  &events.Command{Environment: "test"},
	}
	applyExecutor := &events.ApplyExecutor{
		Workspace: w,
	}
	When(w.GetWorkspace(ctx.BaseRepo, ctx.Pull, ctx.Command.Environment)).ThenReturn("", errors.New("err"))
	response := applyExecutor.Execute(ctx)
	Equals(t, "No workspace found. Did you run plan?", response.Failure)
}

func TestExecute_NoPlansFoundFailure(t *testing.T) {
	t.Log("If no plans are found for an environment we are returning an failure")

	g := ghMocks.NewMockClient()
	w := eventMocks.NewMockWorkspace()
	ctx := &events.CommandContext{
		BaseRepo: models.Repo{FullName: "owner/repo-name"},
		Pull:     models.PullRequest{},
		Command:  &events.Command{Environment: "test"},
		Log:      logging.NewNoopLogger(),
	}
	// Create a temporary directory so we don't iterate over an entire directory
	dir, _ := ioutil.TempDir("", "example-dir")
	defer os.RemoveAll(dir) // clean up
	applyExecutor := &events.ApplyExecutor{
		Github:          g,
		RequireApproval: false,
		Workspace:       w,
	}
	When(w.GetWorkspace(ctx.BaseRepo, ctx.Pull, ctx.Command.Environment)).ThenReturn(dir, nil)
	response := applyExecutor.Execute(ctx)
	Equals(t, "No plans found for that environment.", response.Failure)
}
