package events_test

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hootsuite/atlantis/server/events"
	ghmocks "github.com/hootsuite/atlantis/server/events/github/mocks"
	eventmocks "github.com/hootsuite/atlantis/server/events/mocks"
	"github.com/hootsuite/atlantis/server/events/models"
	rmocks "github.com/hootsuite/atlantis/server/events/run/mocks"
	tmocks "github.com/hootsuite/atlantis/server/events/terraform/mocks"
	"github.com/hootsuite/atlantis/server/logging"
	. "github.com/hootsuite/atlantis/testing"
	. "github.com/petergtz/pegomock"
)

var applyCtx = events.CommandContext{
	Command: &events.Command{
		Name:        events.Apply,
		Environment: "env",
	},
	Log:      logging.NewNoopLogger(),
	BaseRepo: models.Repo{},
	HeadRepo: models.Repo{},
	Pull:     models.PullRequest{},
	User: models.User{
		Username: "anubhavmishra",
	},
}

func TestExecute_RequireApprovalError(t *testing.T) {
	t.Log("If checking whether pull request is approved there is an error we are returning it")

	g := ghmocks.NewMockClient()
	applyExecutor := &events.ApplyExecutor{
		Github:          g,
		RequireApproval: true,
	}
	When(g.PullIsApproved(applyCtx.BaseRepo, applyCtx.Pull)).ThenReturn(false, errors.New("error"))
	response := applyExecutor.Execute(&applyCtx)
	Equals(t, "checking if pull request was approved: error", response.Error.Error())
}

func TestExecute_RequireApprovalIfApproved(t *testing.T) {
	t.Log("If the pull request is not approved there is a failure and we are returning it")

	g := ghmocks.NewMockClient()
	applyExecutor := &events.ApplyExecutor{
		Github:          g,
		RequireApproval: true,
	}
	When(g.PullIsApproved(applyCtx.BaseRepo, applyCtx.Pull)).ThenReturn(false, nil)
	response := applyExecutor.Execute(&applyCtx)
	Equals(t, "Pull request must be approved before running apply.", response.Failure)
}

func TestExecute_GetWorkspaceError(t *testing.T) {
	t.Log("If while getting workspace there is an error we should return a failure")

	w := eventmocks.NewMockWorkspace()
	applyExecutor := &events.ApplyExecutor{
		Workspace: w,
	}
	When(w.GetWorkspace(applyCtx.BaseRepo, applyCtx.Pull, applyCtx.Command.Environment)).ThenReturn("", errors.New("err"))
	response := applyExecutor.Execute(&applyCtx)
	Equals(t, "No workspace found. Did you run plan?", response.Failure)
}

func TestExecute_NoPlansFoundFailure(t *testing.T) {
	t.Log("If no plans are found for an environment we are returning an failure")

	g := ghmocks.NewMockClient()
	w := eventmocks.NewMockWorkspace()
	// Create a temporary directory so we don't iterate over an entire directory
	dir, _ := ioutil.TempDir("", "example-dir")
	defer os.RemoveAll(dir) // clean up
	applyExecutor := &events.ApplyExecutor{
		Github:          g,
		RequireApproval: false,
		Workspace:       w,
	}
	When(w.GetWorkspace(applyCtx.BaseRepo, applyCtx.Pull, applyCtx.Command.Environment)).ThenReturn(dir, nil)
	response := applyExecutor.Execute(&applyCtx)
	Equals(t, "No plans found for that environment.", response.Failure)
}

func TestExecute_ApplyPreExecuteResult(t *testing.T) {
	a, _ := setupApplyExecutorTest(t)
	// Create a temporary directory so we don't iterate over an entire directory
	dir, _ := ioutil.TempDir("", "example-dir")
	defer os.RemoveAll(dir) // clean up
	When(a.Workspace.GetWorkspace(planCtx.BaseRepo, planCtx.Pull, planCtx.Command.Environment)).ThenReturn(
		dir, nil,
	)
	projectResult := events.ProjectResult{
		Failure: "failure",
	}
	When(a.ProjectPreExecute.Execute(&planCtx, dir, models.Project{RepoFullName: "", Path: "."})).
		ThenReturn(events.PreExecuteResult{ProjectResult: projectResult})
	r := a.Execute(&planCtx)
	t.Logf("I am here %s", r)
	// This should be changed to one result
	Assert(t, len(r.ProjectResults) == 0, "exp one project result")
}

func setupApplyExecutorTest(t *testing.T) (*events.ApplyExecutor, *tmocks.MockRunner) {
	RegisterMockTestingT(t)
	gh := ghmocks.NewMockClient()
	w := eventmocks.NewMockWorkspace()
	ppe := eventmocks.NewMockProjectPreExecutor()
	runner := tmocks.NewMockRunner()
	run := rmocks.NewMockRunner()
	a := events.ApplyExecutor{
		Github:            gh,
		Terraform:         runner,
		RequireApproval:   false,
		Run:               run,
		Workspace:         w,
		ProjectPreExecute: ppe,
	}
	return &a, runner
}
