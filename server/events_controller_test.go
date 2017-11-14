package server_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
	"github.com/hootsuite/atlantis/server"
	"github.com/hootsuite/atlantis/server/events"
	emocks "github.com/hootsuite/atlantis/server/events/mocks"
	"github.com/hootsuite/atlantis/server/events/vcs"
	"github.com/hootsuite/atlantis/server/logging"
	"github.com/hootsuite/atlantis/server/mocks"
	//. "github.com/hootsuite/atlantis/testing"
	"time"

	"github.com/hootsuite/atlantis/server/events/models"
	. "github.com/petergtz/pegomock"
)

const secret = "secret"
const githubHeader = "X-Github-Event"
const gitlabHeader = "X-Gitlab-Event"

var eventsReq *http.Request

func TestPost_NotGithubOrGitlab(t *testing.T) {
	t.Log("when the request is not for gitlab or github a 400 is returned")
	e, _, _, _, _, _ := setup(t)
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "Ignoring request")
}

func TestPost_UnsupportedVCSGithub(t *testing.T) {
	t.Log("when the request is for an unsupported vcs a 400 is returned")
	e, _, _, _, _, _ := setup(t)
	e.SupportedVCSHosts = nil
	eventsReq.Header.Set(githubHeader, "value")
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "Ignoring request since not configured to support GitHub")
}

func TestPost_UnsupportedVCSGitlab(t *testing.T) {
	t.Log("when the request is for an unsupported vcs a 400 is returned")
	e, _, _, _, _, _ := setup(t)
	e.SupportedVCSHosts = nil
	eventsReq.Header.Set(gitlabHeader, "value")
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "Ignoring request since not configured to support GitLab")
}

func TestPost_InvalidGithubSecret(t *testing.T) {
	t.Log("when the github payload can't be validated a 400 is returned")
	e, v, _, _, _, _ := setup(t)
	w := httptest.NewRecorder()
	eventsReq.Header.Set(githubHeader, "value")
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn(nil, errors.New("err"))
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "err")
}

func TestPost_InvalidGitlabSecret(t *testing.T) {
	t.Log("when the gitlab payload can't be validated a 400 is returned")
	e, _, gl, _, _, _ := setup(t)
	w := httptest.NewRecorder()
	eventsReq.Header.Set(gitlabHeader, "value")
	When(gl.Validate(eventsReq, []byte(secret))).ThenReturn(nil, errors.New("err"))
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "err")
}

func TestPost_UnsupportedGithubEvent(t *testing.T) {
	t.Log("when the event type is an unsupported github event we ignore it")
	e, v, _, _, _, _ := setup(t)
	w := httptest.NewRecorder()
	eventsReq.Header.Set(githubHeader, "value")
	When(v.Validate(eventsReq, nil)).ThenReturn([]byte(`{"not an event": ""}`), nil)
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusOK, "Ignoring unsupported event")
}

func TestPost_UnsupportedGitlabEvent(t *testing.T) {
	t.Log("when the event type is an unsupported gitlab event we ignore it")
	e, _, gl, _, _, _ := setup(t)
	w := httptest.NewRecorder()
	eventsReq.Header.Set(gitlabHeader, "value")
	When(gl.Validate(eventsReq, nil)).ThenReturn([]byte(`{"not an event": ""}`), nil)
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusOK, "Ignoring unsupported event")
}

func TestPost_GithubCommentNotCreated(t *testing.T) {
	t.Log("when the event is a github comment but it's not a created event we ignore it")
	e, v, _, _, _, _ := setup(t)
	eventsReq.Header.Set(githubHeader, "issue_comment")
	// comment action is deleted, not created
	event := `{"action": "deleted"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusOK, "Ignoring comment event since action was not created")
}

func TestPost_GithubInvalidComment(t *testing.T) {
	t.Log("when the event is a github comment without all expected data we return a 400")
	e, v, _, p, _, _ := setup(t)
	eventsReq.Header.Set(githubHeader, "issue_comment")
	event := `{"action": "created"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	When(p.ParseGithubIssueCommentEvent(AnyComment())).ThenReturn(models.Repo{}, models.User{}, 1, errors.New("err"))
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "Failed parsing event")
}

func TestPost_GithubCommentInvalidCommand(t *testing.T) {
	t.Log("when the event is a github comment with an invalid command we ignore it")
	e, v, _, p, _, _ := setup(t)
	eventsReq.Header.Set(githubHeader, "issue_comment")
	event := `{"action": "created"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	When(p.ParseGithubIssueCommentEvent(AnyComment())).ThenReturn(models.Repo{}, models.User{}, 1, nil)
	When(p.DetermineCommand("", vcs.Github)).ThenReturn(nil, errors.New("err"))
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusOK, "Ignoring: err")
}

func TestPost_GithubCommentSuccess(t *testing.T) {
	t.Log("when the event is a github comment with a valid command we call the command handler")
	e, v, _, p, cr, _ := setup(t)
	eventsReq.Header.Set(githubHeader, "issue_comment")
	event := `{"action": "created"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	baseRepo := models.Repo{}
	user := models.User{}
	cmd := events.Command{}
	When(p.ParseGithubIssueCommentEvent(AnyComment())).ThenReturn(baseRepo, user, 1, nil)
	When(p.DetermineCommand("", vcs.Github)).ThenReturn(&cmd, nil)
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusOK, "Processing...")

	// wait for 200ms so goroutine is called
	time.Sleep(200 * time.Millisecond)
	cr.VerifyWasCalledOnce().ExecuteGithubCommand(baseRepo, user, 1, &cmd)
}

func TestPost_GithubPullRequestNotClosed(t *testing.T) {
	t.Log("when the event is a github pull reuqest but it's not a closed event we ignore it")
	e, v, _, _, _, _ := setup(t)
	eventsReq.Header.Set(githubHeader, "pull_request")
	event := `{"action": "opened"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusOK, "Ignoring pull request event since action was not closed")
}

func TestPost_GithubPullRequestInvalid(t *testing.T) {
	t.Log("when the event is a github pull request with invalid data we return a 400")
	e, v, _, p, _, _ := setup(t)
	eventsReq.Header.Set(githubHeader, "pull_request")

	event := `{"action": "closed"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	When(p.ParseGithubPull(AnyPull())).ThenReturn(models.PullRequest{}, models.Repo{}, errors.New("err"))
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "Error parsing pull data: err")
}

func TestPost_GithubPullRequestInvalidRepo(t *testing.T) {
	t.Log("when the event is a github pull request with invalid repo data we return a 400")
	e, v, _, p, _, _ := setup(t)
	eventsReq.Header.Set(githubHeader, "pull_request")

	event := `{"action": "closed"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	When(p.ParseGithubPull(AnyPull())).ThenReturn(models.PullRequest{}, models.Repo{}, nil)
	When(p.ParseGithubRepo(AnyRepo())).ThenReturn(models.Repo{}, errors.New("err"))
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusBadRequest, "Error parsing repo data: err")
}

func TestPost_GithubPullRequestErrCleaningPull(t *testing.T) {
	t.Log("when the event is a pull request and we have an error calling CleanUpPull we return a 503")
	RegisterMockTestingT(t)
	e, v, _, p, _, c := setup(t)
	eventsReq.Header.Set(githubHeader, "pull_request")

	event := `{"action": "closed"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	repo := models.Repo{}
	pull := models.PullRequest{}
	When(p.ParseGithubPull(AnyPull())).ThenReturn(pull, repo, nil)
	When(p.ParseGithubRepo(AnyRepo())).ThenReturn(repo, nil)
	When(c.CleanUpPull(repo, pull, vcs.Github)).ThenReturn(errors.New("cleanup err"))
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusInternalServerError, "Error cleaning pull request: cleanup err")
}

func TestPost_GithubPullRequestSuccess(t *testing.T) {
	t.Log("when the event is a pull request and everything works we return a 200")
	e, v, _, p, _, c := setup(t)
	eventsReq.Header.Set(githubHeader, "pull_request")

	event := `{"action": "closed"}`
	When(v.Validate(eventsReq, []byte(secret))).ThenReturn([]byte(event), nil)
	repo := models.Repo{}
	pull := models.PullRequest{}
	When(p.ParseGithubPull(AnyPull())).ThenReturn(pull, repo, nil)
	When(p.ParseGithubRepo(AnyRepo())).ThenReturn(repo, nil)
	When(c.CleanUpPull(repo, pull, vcs.Github)).ThenReturn(nil)
	w := httptest.NewRecorder()
	e.Post(w, eventsReq)
	responseContains(t, w, http.StatusOK, "Pull request cleaned successfully")
}

func setup(t *testing.T) (server.EventsController, *mocks.MockGHRequestValidator, *mocks.MockGitlabRequestParser, *emocks.MockEventParsing, *emocks.MockCommandRunner, *emocks.MockPullCleaner) {
	RegisterMockTestingT(t)
	eventsReq, _ = http.NewRequest("GET", "", bytes.NewBuffer(nil))
	v := mocks.NewMockGHRequestValidator()
	gl := mocks.NewMockGitlabRequestParser()
	p := emocks.NewMockEventParsing()
	cr := emocks.NewMockCommandRunner()
	c := emocks.NewMockPullCleaner()
	e := server.EventsController{
		Logger:                 logging.NewNoopLogger(),
		GithubRequestValidator: v,
		Parser:                 p,
		CommandRunner:          cr,
		PullCleaner:            c,
		GithubWebHookSecret:    []byte(secret),
		SupportedVCSHosts:      []vcs.Host{vcs.Github, vcs.Gitlab},
		GitlabWebHookSecret:    []byte(secret),
		GitlabRequestParser:    gl,
	}
	return e, v, gl, p, cr, c
}

func AnyComment() *github.IssueCommentEvent {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf(&github.IssueCommentEvent{})))
	return &github.IssueCommentEvent{}
}

func AnyPull() *github.PullRequest {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf(&github.PullRequest{})))
	return &github.PullRequest{}
}

func AnyRepo() *github.Repository {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf(&github.Repository{})))
	return &github.Repository{}
}

func AnyCommandContext() *events.CommandContext {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf(&events.CommandContext{})))
	return &events.CommandContext{}
}
