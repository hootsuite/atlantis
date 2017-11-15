package webhooks_test

import (
	"regexp"
	"testing"

	"github.com/hootsuite/atlantis/server/events/webhooks"
	"github.com/hootsuite/atlantis/server/events/webhooks/mocks"
	. "github.com/hootsuite/atlantis/testing"
	. "github.com/petergtz/pegomock"
)

func TestSend_Success(t *testing.T) {
	t.Log("Sending a hook with a matching regex should call PostMessage")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	regex, err := regexp.Compile(".*")
	Ok(t, err)

	channel := "somechannel"
	hook := webhooks.SlackWebhook{
		Client:   client,
		EnvRegex: regex,
		Channel:  channel,
	}
	result := webhooks.ApplyResult{
		Environment: "production",
	}
	hook.Send(result)
	client.VerifyWasCalledOnce().PostMessage(channel, result)
}

func TestSend_NoopSuccess(t *testing.T) {
	t.Log("Sending a hook with a non-matching regex should succeed")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	regex, err := regexp.Compile("weirdemv")
	Ok(t, err)

	hook := webhooks.SlackWebhook{
		Client:   client,
		EnvRegex: regex,
		Channel:  "somechannel",
	}
	result := webhooks.ApplyResult{
		Environment: "production",
	}
	err = hook.Send(result)
	Ok(t, err)
}
