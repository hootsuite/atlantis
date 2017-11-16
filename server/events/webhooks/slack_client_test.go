package webhooks_test

import (
	"testing"

	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/events/webhooks"

	. "github.com/hootsuite/atlantis/testing"
)

const invalidSlackToken = "invalidtoken"

func TestNewSlackClient_Success(t *testing.T) {
	t.Log("NewSlackClient should always return a non-nil client")
	client := webhooks.NewSlackClient(invalidSlackToken)
	Assert(t, client != nil, "SlackClient shouldn't be nil")

	client = webhooks.NewSlackClient("")
	Assert(t, client != nil, "SlackClient shouldn't be nil")
}

func TestAuthTest_Error(t *testing.T) {
	t.Log("When a SlackClient is created with an invalid token, AuthTest should error")
	client := webhooks.NewSlackClient(invalidSlackToken)
	err := client.AuthTest()
	Assert(t, err != nil, "expected error")
}

func TestChannelExists_Error(t *testing.T) {
	t.Log("When a SlackClient is created with an invalid token, ChannelExists should error")
	client := webhooks.NewSlackClient(invalidSlackToken)
	_, err := client.ChannelExists("somechannel")
	Assert(t, err != nil, "expected error")
}

func TestPostMessage_Error(t *testing.T) {
	t.Log("When a SlackClient is created with an invalid token, PostMessage should error")
	client := webhooks.NewSlackClient(invalidSlackToken)
	// todo: ?make this ApplyResult a fixture
	result := webhooks.ApplyResult{
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
	err := client.PostMessage("somechannel", result)
	Assert(t, err != nil, "expected error")
}

func TestTokenIsSet(t *testing.T) {
	client := webhooks.NewSlackClient(invalidSlackToken)
}
