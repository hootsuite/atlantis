package events_test

import (
	"bytes"
	"errors"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hootsuite/atlantis/server/events"
	"github.com/hootsuite/atlantis/server/events/mocks"
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/events/models/fixtures"
	"github.com/hootsuite/atlantis/server/events/vcs"
	vcsmocks "github.com/hootsuite/atlantis/server/events/vcs/mocks"
	logmocks "github.com/hootsuite/atlantis/server/logging/mocks"
	. "github.com/hootsuite/atlantis/testing"
	. "github.com/petergtz/pegomock"
)

var applier *mocks.MockExecutor
var helper *mocks.MockExecutor
var planner *mocks.MockExecutor
var eventParsing *mocks.MockEventParsing
var vcsClient *vcsmocks.MockClientProxy
var ghStatus *mocks.MockCommitStatusUpdater
var githubGetter *mocks.MockGithubPullGetter
var gitlabGetter *mocks.MockGitlabMergeRequestGetter
var envLocker *mocks.MockEnvLocker
var ch events.CommandHandler
var logBytes *bytes.Buffer

func setup(t *testing.T) {
	RegisterMockTestingT(t)
	applier = mocks.NewMockExecutor()
	helper = mocks.NewMockExecutor()
	planner = mocks.NewMockExecutor()
	eventParsing = mocks.NewMockEventParsing()
	ghStatus = mocks.NewMockCommitStatusUpdater()
	envLocker = mocks.NewMockEnvLocker()
	vcsClient = vcsmocks.NewMockClientProxy()
	githubGetter = mocks.NewMockGithubPullGetter()
	gitlabGetter = mocks.NewMockGitlabMergeRequestGetter()
	logger := logmocks.NewMockSimpleLogging()
	logBytes = new(bytes.Buffer)
	When(logger.Underlying()).ThenReturn(log.New(logBytes, "", 0))
	ch = events.CommandHandler{
		PlanExecutor:             planner,
		ApplyExecutor:            applier,
		HelpExecutor:             helper,
		VCSClient:                vcsClient,
		GHStatus:                 ghStatus,
		EventParser:              eventParsing,
		EnvLocker:                envLocker,
		GHCommentRenderer:        &events.GithubCommentRenderer{},
		GithubPullGetter:         githubGetter,
		GitlabMergeRequestGetter: gitlabGetter,
		Logger: logger,
	}
}

func TestExecuteCommand_LogPanics(t *testing.T) {
	t.Log("if there is a panic it is commented back on the pull request")
	setup(t)
	When(ghStatus.Update(fixtures.Repo, fixtures.Pull, vcs.Pending, nil, vcs.Github)).ThenPanic("panic")
	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, 1, nil, vcs.Github)
	_, _, comment, _ := vcsClient.VerifyWasCalledOnce().CreateComment(AnyRepo(), AnyPullRequest(), AnyString(), AnyVCSHost()).GetCapturedArguments()
	Assert(t, strings.Contains(comment, "Error: goroutine panic"), "comment should be about a goroutine panic")
}

func TestExecuteCommand_NoGithubPullGetter(t *testing.T) {
	t.Log("if CommandHandler was constructed with a nil GithubPullGetter an error should be logged")
	setup(t)
	ch.GithubPullGetter = nil
	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, 1, nil, vcs.Github)
	Equals(t, "[ERROR] hootsuite/atlantis#1: Atlantis not configured to support GitHub\n", logBytes.String())
}

func TestExecuteCommand_NoGitlabMergeGetter(t *testing.T) {
	t.Log("if CommandHandler was constructed with a nil GitlabMergeRequestGetter an error should be logged")
	setup(t)
	ch.GitlabMergeRequestGetter = nil
	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, 1, nil, vcs.Gitlab)
	Equals(t, "[ERROR] hootsuite/atlantis#1: Atlantis not configured to support GitLab\n", logBytes.String())
}

func TestExecuteCommand_GithubPullErr(t *testing.T) {
	t.Log("if getting the github pull request fails an error should be logged")
	setup(t)
	When(githubGetter.GetPullRequest(fixtures.Repo, fixtures.Pull.Num)).ThenReturn(nil, errors.New("err"))
	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, fixtures.Pull.Num, nil, vcs.Github)
	Equals(t, "[ERROR] hootsuite/atlantis#1: Making pull request API call to GitHub: err\n", logBytes.String())
}

func TestExecuteCommand_GitlabMergeRequestErr(t *testing.T) {
	t.Log("if getting the gitlab merge request fails an error should be logged")
	setup(t)
	When(gitlabGetter.GetMergeRequest(fixtures.Repo.FullName, fixtures.Pull.Num)).ThenReturn(nil, errors.New("err"))
	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, fixtures.Pull.Num, nil, vcs.Gitlab)
	Equals(t, "[ERROR] hootsuite/atlantis#1: Making merge request API call to GitLab: err\n", logBytes.String())
}

func TestExecuteCommand_GithubPullParseErr(t *testing.T) {
	t.Log("if parsing the returned github pull request fails an error should be logged")
	setup(t)
	var pull github.PullRequest
	When(githubGetter.GetPullRequest(fixtures.Repo, fixtures.Pull.Num)).ThenReturn(&pull, nil)
	When(eventParsing.ParseGithubPull(&pull)).ThenReturn(fixtures.Pull, fixtures.Repo, errors.New("err"))

	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, fixtures.Pull.Num, nil, vcs.Github)
	Equals(t, "[ERROR] hootsuite/atlantis#1: Extracting required fields from comment data: err\n", logBytes.String())
}

func TestExecuteCommand_ClosedPull(t *testing.T) {
	t.Log("if a command is run on a closed pull request atlantis should" +
		" comment saying that this is not allowed")
	setup(t)
	pull := &github.PullRequest{
		State: github.String("closed"),
	}
	modelPull := models.PullRequest{State: models.Closed}
	When(githubGetter.GetPullRequest(fixtures.Repo, fixtures.Pull.Num)).ThenReturn(pull, nil)
	When(eventParsing.ParseGithubPull(pull)).ThenReturn(modelPull, fixtures.Repo, nil)

	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, fixtures.Pull.Num, nil, vcs.Github)
	vcsClient.VerifyWasCalledOnce().CreateComment(fixtures.Repo, modelPull, "Atlantis commands can't be run on closed pull requests", vcs.Github)
}

func TestExecuteCommand_EnvLocked(t *testing.T) {
	t.Log("if the environment is locked, should comment back on the pull")
	setup(t)
	pull := &github.PullRequest{
		State: github.String("closed"),
	}
	cmd := events.Command{
		Name:        events.Plan,
		Environment: "env",
	}

	When(githubGetter.GetPullRequest(fixtures.Repo, fixtures.Pull.Num)).ThenReturn(pull, nil)
	When(eventParsing.ParseGithubPull(pull)).ThenReturn(fixtures.Pull, fixtures.Repo, nil)
	When(envLocker.TryLock(fixtures.Repo.FullName, cmd.Environment, fixtures.Pull.Num)).ThenReturn(false)
	ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, fixtures.Pull.Num, &cmd, vcs.Github)

	msg := "The env environment is currently locked by another" +
		" command that is running for this pull request." +
		" Wait until the previous command is complete and try again."
	ghStatus.VerifyWasCalledOnce().Update(fixtures.Repo, fixtures.Pull, vcs.Pending, &cmd, vcs.Github)
	_, response := ghStatus.VerifyWasCalledOnce().UpdateProjectResult(AnyCommandContext(), AnyCommandResponse()).GetCapturedArguments()
	Equals(t, msg, response.Failure)
	vcsClient.VerifyWasCalledOnce().CreateComment(fixtures.Repo, fixtures.Pull,
		"**Plan Failed**: "+msg+"\n\n", vcs.Github)
}

func TestExecuteCommand_FullRun(t *testing.T) {
	t.Log("when running a plan, apply or help should comment")
	pull := &github.PullRequest{
		State: github.String("closed"),
	}
	cmdResponse := events.CommandResponse{}
	for _, c := range []events.CommandName{events.Help, events.Plan, events.Apply} {
		setup(t)
		cmd := events.Command{
			Name:        c,
			Environment: "env",
		}
		When(githubGetter.GetPullRequest(fixtures.Repo, fixtures.Pull.Num)).ThenReturn(pull, nil)
		When(eventParsing.ParseGithubPull(pull)).ThenReturn(fixtures.Pull, fixtures.Repo, nil)
		When(envLocker.TryLock(fixtures.Repo.FullName, cmd.Environment, fixtures.Pull.Num)).ThenReturn(true)
		switch c {
		case events.Help:
			When(helper.Execute(AnyCommandContext())).ThenReturn(cmdResponse)
		case events.Plan:
			When(planner.Execute(AnyCommandContext())).ThenReturn(cmdResponse)
		case events.Apply:
			When(applier.Execute(AnyCommandContext())).ThenReturn(cmdResponse)
		}

		ch.ExecuteCommand(fixtures.Repo, fixtures.Repo, fixtures.User, fixtures.Pull.Num, &cmd, vcs.Github)

		ghStatus.VerifyWasCalledOnce().Update(fixtures.Repo, fixtures.Pull, vcs.Pending, &cmd, vcs.Github)
		_, response := ghStatus.VerifyWasCalledOnce().UpdateProjectResult(AnyCommandContext(), AnyCommandResponse()).GetCapturedArguments()
		Equals(t, cmdResponse, response)
		vcsClient.VerifyWasCalledOnce().CreateComment(AnyRepo(), AnyPullRequest(), AnyString(), AnyVCSHost())
		envLocker.VerifyWasCalledOnce().Unlock(fixtures.Repo.FullName, cmd.Environment, fixtures.Pull.Num)
	}
}

func AnyCommandContext() *events.CommandContext {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf(&events.CommandContext{})))
	return &events.CommandContext{}
}

func AnyVCSHost() vcs.Host {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf(vcs.Github)))
	return vcs.Github
}

func AnyCommandResponse() events.CommandResponse {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf(events.CommandResponse{})))
	return events.CommandResponse{}
}
