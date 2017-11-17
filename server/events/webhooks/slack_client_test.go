package webhooks_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/events/webhooks"
	"github.com/hootsuite/atlantis/server/events/webhooks/mocks"
	"github.com/nlopes/slack"

	. "github.com/hootsuite/atlantis/testing"
	. "github.com/petergtz/pegomock"
)

var wrapper *mocks.MockSlackWrapper
var client webhooks.DefaultSlackClient
var result webhooks.ApplyResult

func setup(t *testing.T) {
	RegisterMockTestingT(t)
	wrapper = mocks.NewMockSlackWrapper()
	client = webhooks.DefaultSlackClient{
		Slack: wrapper,
		Token: "sometoken",
	}
	result = webhooks.ApplyResult{
		Workspace: "production",
		Repo: models.Repo{
			CloneURL:          "https://user:password@github.com/hootsuite/atlantis.git",
			FullName:          "hootsuite/atlantis",
			Owner:             "hootsuite",
			SanitizedCloneURL: "https://github.com/hootsuite/atlantis.git",
			Name:              "atlantis",
		},
		Pull: models.PullRequest{
			Num:        1,
			HeadCommit: "16ca62f65c18ff456c6ef4cacc8d4826e264bb17",
			Branch:     "branch",
			Author:     "lkysow",
			URL:        "url",
		},
		User: models.User{
			Username: "lkysow",
		},
		Success: true,
	}
}

func TestNewSlackClient(t *testing.T) {
	t.Log("NewSlackClient should always return a non-nil client")
	c := webhooks.NewSlackClient("invalidSlackToken")
	Assert(t, c != nil, "SlackClient shouldn't be nil")

	c = webhooks.NewSlackClient("")
	Assert(t, c != nil, "SlackClient shouldn't be nil")
}

func TestAuthTest_Success(t *testing.T) {
	t.Log("When the underylying client suceeds, function should succeed")
	setup(t)
	err := client.AuthTest()
	Ok(t, err)
}

func TestAuthTest_Error(t *testing.T) {
	t.Log("When the underylying slack client errors, an error should be returned")
	setup(t)
	When(wrapper.AuthTest()).ThenReturn(nil, errors.New(""))
	err := client.AuthTest()
	Assert(t, err != nil, "expected error")
}

func TestTokenIsSet(t *testing.T) {
	t.Log("When the Token is an empty string, function should return false")
	c := webhooks.DefaultSlackClient{
		Token: "",
	}
	Equals(t, false, c.TokenIsSet())

	t.Log("When the Token is not an empty string, function should return true")
	c.Token = "random"
	Equals(t, true, c.TokenIsSet())
}

func TestChannelExists_False(t *testing.T) {
	t.Log("When the slack channel doesn't exist, function should return false")
	setup(t)
	When(wrapper.GetChannels(true)).ThenReturn([]slack.Channel{}, nil)

	exists, err := client.ChannelExists("somechannel")
	Ok(t, err)
	Equals(t, false, exists)
}

func TestChannelExists_True(t *testing.T) {
	t.Log("When the slack channel exists, function should return true")
	setup(t)
	channelJSON := `{"name":"existingchannel"}`
	var channel slack.Channel
	err := json.Unmarshal([]byte(channelJSON), &channel)
	Ok(t, err)

	When(wrapper.GetChannels(true)).ThenReturn([]slack.Channel{channel}, nil)

	exists, err := client.ChannelExists("existingchannel")
	Ok(t, err)
	Equals(t, true, exists)
}

func TestChannelExists_Error(t *testing.T) {
	t.Log("When the underylying slack client errors, an error should be returned")
	setup(t)
	When(wrapper.GetChannels(true)).ThenReturn(nil, errors.New(""))

	_, err := client.ChannelExists("anychannel")
	Assert(t, err != nil, "expected error")
}

func TestPostMessage_Success(t *testing.T) {
	t.Log("When apply succeds, function should succeed and indicate success")
	setup(t)

	expParams := slack.NewPostMessageParameters()
	expParams.Attachments = []slack.Attachment{{
		Color: "good",
		Text:  "Apply succeeded for <url|hootsuite/atlantis>",
		Fields: []slack.AttachmentField{
			{
				Title: "Workspace",
				Value: result.Workspace,
				Short: true,
			},
			{
				Title: "User",
				Value: result.User.Username,
				Short: true,
			},
		},
	}}
	expParams.AsUser = true
	expParams.EscapeText = false

	channel := "somechannel"
	err := client.PostMessage(channel, result)
	Ok(t, err)
	wrapper.VerifyWasCalledOnce().PostMessage(channel, "", expParams)

	t.Log("When apply fails, function should succeed and indicate failure")
	result.Success = false
	expParams.Attachments[0].Color = "danger"
	expParams.Attachments[0].Text = "Apply failed for <url|hootsuite/atlantis>"

	err = client.PostMessage(channel, result)
	Ok(t, err)
	wrapper.VerifyWasCalledOnce().PostMessage(channel, "", expParams)
}

func TestPostMessage_Error(t *testing.T) {
	t.Log("When the underylying slack client errors, an error should be returned")
	setup(t)

	expParams := slack.NewPostMessageParameters()
	expParams.Attachments = []slack.Attachment{{
		Color: "good",
		Text:  "Apply succeeded for <url|hootsuite/atlantis>",
		Fields: []slack.AttachmentField{
			{
				Title: "Workspace",
				Value: result.Workspace,
				Short: true,
			},
			{
				Title: "User",
				Value: result.User.Username,
				Short: true,
			},
		},
	}}
	expParams.AsUser = true
	expParams.EscapeText = false

	channel := "somechannel"
	When(wrapper.PostMessage(channel, "", expParams)).ThenReturn("", "", errors.New(""))

	err := client.PostMessage(channel, result)
	Assert(t, err != nil, "expected error")
}
