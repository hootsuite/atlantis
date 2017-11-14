package webhooks_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hootsuite/atlantis/server/events/webhooks"
	"github.com/hootsuite/atlantis/server/events/webhooks/mocks"
)

func TestNewWebhooksManager_InvalidToken(t *testing.T) {
	t.Log("When given an empty slack token and there is a slack webhook config, an error is returned")
	emptyToken := ""
	_, err := webhooks.NewWebhooksManager(validConfigs(), webhooks.NewClient(emptyToken))
	Assert(t, err != nil, "expected error")
	// Equals(t, "for slack webhooks, slack-token must be set", err.Error())
	Assert(t, strings.Contains(err.Error(), "testing slack authentication"), "expected auth error")
}

func TestSend_Success(t *testing.T) {
	RegisterMockTestingT(t)
	client := mocks.MockSlackClient()
	regex, err := regexp.Compile(".*")
	Ok(t, err)
	hook := SlackWebhook{
		Client:   client,
		EnvRegex: regex,
		Channel:  "somechannel",
	}
	result := webhooks.ApplyResult{
		Environment: "production"
	}
	hook.Send(result)
}

func TestSend_Error(t *testing.T) {

}
