// Automatically generated by pegomock. DO NOT EDIT!
// Source: github.com/hootsuite/atlantis/server/events (interfaces: EventParsing)

package mocks

import (
	"reflect"

	github "github.com/google/go-github/github"
	events "github.com/hootsuite/atlantis/server/events"
	models "github.com/hootsuite/atlantis/server/events/models"
	pegomock "github.com/petergtz/pegomock"
)

type MockEventParsing struct {
	fail func(message string, callerSkip ...int)
}

func NewMockEventParsing() *MockEventParsing {
	return &MockEventParsing{fail: pegomock.GlobalFailHandler}
}

func (mock *MockEventParsing) DetermineCommand(comment *github.IssueCommentEvent) (*events.Command, error) {
	params := []pegomock.Param{comment}
	result := pegomock.GetGenericMockFrom(mock).Invoke("DetermineCommand", params, []reflect.Type{reflect.TypeOf((**events.Command)(nil)).Elem(), reflect.TypeOf((*error)(nil)).Elem()})
	var ret0 *events.Command
	var ret1 error
	if len(result) != 0 {
		if result[0] != nil {
			ret0 = result[0].(*events.Command)
		}
		if result[1] != nil {
			ret1 = result[1].(error)
		}
	}
	return ret0, ret1
}

func (mock *MockEventParsing) ExtractCommentData(comment *github.IssueCommentEvent) (models.Repo, models.User, models.PullRequest, error) {
	params := []pegomock.Param{comment}
	result := pegomock.GetGenericMockFrom(mock).Invoke("ParseGithubIssueCommentEvent", params, []reflect.Type{reflect.TypeOf((*models.Repo)(nil)).Elem(), reflect.TypeOf((*models.User)(nil)).Elem(), reflect.TypeOf((*models.PullRequest)(nil)).Elem(), reflect.TypeOf((*error)(nil)).Elem()})
	var ret0 models.Repo
	var ret1 models.User
	var ret2 models.PullRequest
	var ret3 error
	if len(result) != 0 {
		if result[0] != nil {
			ret0 = result[0].(models.Repo)
		}
		if result[1] != nil {
			ret1 = result[1].(models.User)
		}
		if result[2] != nil {
			ret2 = result[2].(models.PullRequest)
		}
		if result[3] != nil {
			ret3 = result[3].(error)
		}
	}
	return ret0, ret1, ret2, ret3
}

func (mock *MockEventParsing) ExtractPullData(pull *github.PullRequest) (models.PullRequest, models.Repo, error) {
	params := []pegomock.Param{pull}
	result := pegomock.GetGenericMockFrom(mock).Invoke("ParseGithubPull", params, []reflect.Type{reflect.TypeOf((*models.PullRequest)(nil)).Elem(), reflect.TypeOf((*models.Repo)(nil)).Elem(), reflect.TypeOf((*error)(nil)).Elem()})
	var ret0 models.PullRequest
	var ret1 models.Repo
	var ret2 error
	if len(result) != 0 {
		if result[0] != nil {
			ret0 = result[0].(models.PullRequest)
		}
		if result[1] != nil {
			ret1 = result[1].(models.Repo)
		}
		if result[2] != nil {
			ret2 = result[2].(error)
		}
	}
	return ret0, ret1, ret2
}

func (mock *MockEventParsing) ExtractRepoData(ghRepo *github.Repository) (models.Repo, error) {
	params := []pegomock.Param{ghRepo}
	result := pegomock.GetGenericMockFrom(mock).Invoke("ParseGithubRepo", params, []reflect.Type{reflect.TypeOf((*models.Repo)(nil)).Elem(), reflect.TypeOf((*error)(nil)).Elem()})
	var ret0 models.Repo
	var ret1 error
	if len(result) != 0 {
		if result[0] != nil {
			ret0 = result[0].(models.Repo)
		}
		if result[1] != nil {
			ret1 = result[1].(error)
		}
	}
	return ret0, ret1
}

func (mock *MockEventParsing) VerifyWasCalledOnce() *VerifierEventParsing {
	return &VerifierEventParsing{mock, pegomock.Times(1), nil}
}

func (mock *MockEventParsing) VerifyWasCalled(invocationCountMatcher pegomock.Matcher) *VerifierEventParsing {
	return &VerifierEventParsing{mock, invocationCountMatcher, nil}
}

func (mock *MockEventParsing) VerifyWasCalledInOrder(invocationCountMatcher pegomock.Matcher, inOrderContext *pegomock.InOrderContext) *VerifierEventParsing {
	return &VerifierEventParsing{mock, invocationCountMatcher, inOrderContext}
}

type VerifierEventParsing struct {
	mock                   *MockEventParsing
	invocationCountMatcher pegomock.Matcher
	inOrderContext         *pegomock.InOrderContext
}

func (verifier *VerifierEventParsing) DetermineCommand(comment *github.IssueCommentEvent) *EventParsing_DetermineCommand_OngoingVerification {
	params := []pegomock.Param{comment}
	methodInvocations := pegomock.GetGenericMockFrom(verifier.mock).Verify(verifier.inOrderContext, verifier.invocationCountMatcher, "DetermineCommand", params)
	return &EventParsing_DetermineCommand_OngoingVerification{mock: verifier.mock, methodInvocations: methodInvocations}
}

type EventParsing_DetermineCommand_OngoingVerification struct {
	mock              *MockEventParsing
	methodInvocations []pegomock.MethodInvocation
}

func (c *EventParsing_DetermineCommand_OngoingVerification) GetCapturedArguments() *github.IssueCommentEvent {
	comment := c.GetAllCapturedArguments()
	return comment[len(comment)-1]
}

func (c *EventParsing_DetermineCommand_OngoingVerification) GetAllCapturedArguments() (_param0 []*github.IssueCommentEvent) {
	params := pegomock.GetGenericMockFrom(c.mock).GetInvocationParams(c.methodInvocations)
	if len(params) > 0 {
		_param0 = make([]*github.IssueCommentEvent, len(params[0]))
		for u, param := range params[0] {
			_param0[u] = param.(*github.IssueCommentEvent)
		}
	}
	return
}

func (verifier *VerifierEventParsing) ExtractCommentData(comment *github.IssueCommentEvent) *EventParsing_ExtractCommentData_OngoingVerification {
	params := []pegomock.Param{comment}
	methodInvocations := pegomock.GetGenericMockFrom(verifier.mock).Verify(verifier.inOrderContext, verifier.invocationCountMatcher, "ParseGithubIssueCommentEvent", params)
	return &EventParsing_ExtractCommentData_OngoingVerification{mock: verifier.mock, methodInvocations: methodInvocations}
}

type EventParsing_ExtractCommentData_OngoingVerification struct {
	mock              *MockEventParsing
	methodInvocations []pegomock.MethodInvocation
}

func (c *EventParsing_ExtractCommentData_OngoingVerification) GetCapturedArguments() *github.IssueCommentEvent {
	comment := c.GetAllCapturedArguments()
	return comment[len(comment)-1]
}

func (c *EventParsing_ExtractCommentData_OngoingVerification) GetAllCapturedArguments() (_param0 []*github.IssueCommentEvent) {
	params := pegomock.GetGenericMockFrom(c.mock).GetInvocationParams(c.methodInvocations)
	if len(params) > 0 {
		_param0 = make([]*github.IssueCommentEvent, len(params[0]))
		for u, param := range params[0] {
			_param0[u] = param.(*github.IssueCommentEvent)
		}
	}
	return
}

func (verifier *VerifierEventParsing) ExtractPullData(pull *github.PullRequest) *EventParsing_ExtractPullData_OngoingVerification {
	params := []pegomock.Param{pull}
	methodInvocations := pegomock.GetGenericMockFrom(verifier.mock).Verify(verifier.inOrderContext, verifier.invocationCountMatcher, "ParseGithubPull", params)
	return &EventParsing_ExtractPullData_OngoingVerification{mock: verifier.mock, methodInvocations: methodInvocations}
}

type EventParsing_ExtractPullData_OngoingVerification struct {
	mock              *MockEventParsing
	methodInvocations []pegomock.MethodInvocation
}

func (c *EventParsing_ExtractPullData_OngoingVerification) GetCapturedArguments() *github.PullRequest {
	pull := c.GetAllCapturedArguments()
	return pull[len(pull)-1]
}

func (c *EventParsing_ExtractPullData_OngoingVerification) GetAllCapturedArguments() (_param0 []*github.PullRequest) {
	params := pegomock.GetGenericMockFrom(c.mock).GetInvocationParams(c.methodInvocations)
	if len(params) > 0 {
		_param0 = make([]*github.PullRequest, len(params[0]))
		for u, param := range params[0] {
			_param0[u] = param.(*github.PullRequest)
		}
	}
	return
}

func (verifier *VerifierEventParsing) ExtractRepoData(ghRepo *github.Repository) *EventParsing_ExtractRepoData_OngoingVerification {
	params := []pegomock.Param{ghRepo}
	methodInvocations := pegomock.GetGenericMockFrom(verifier.mock).Verify(verifier.inOrderContext, verifier.invocationCountMatcher, "ParseGithubRepo", params)
	return &EventParsing_ExtractRepoData_OngoingVerification{mock: verifier.mock, methodInvocations: methodInvocations}
}

type EventParsing_ExtractRepoData_OngoingVerification struct {
	mock              *MockEventParsing
	methodInvocations []pegomock.MethodInvocation
}

func (c *EventParsing_ExtractRepoData_OngoingVerification) GetCapturedArguments() *github.Repository {
	ghRepo := c.GetAllCapturedArguments()
	return ghRepo[len(ghRepo)-1]
}

func (c *EventParsing_ExtractRepoData_OngoingVerification) GetAllCapturedArguments() (_param0 []*github.Repository) {
	params := pegomock.GetGenericMockFrom(c.mock).GetInvocationParams(c.methodInvocations)
	if len(params) > 0 {
		_param0 = make([]*github.Repository, len(params[0]))
		for u, param := range params[0] {
			_param0[u] = param.(*github.Repository)
		}
	}
	return
}
