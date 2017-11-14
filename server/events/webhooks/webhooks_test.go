package webhooks_test

import (
	"strings"
	"testing"

	"github.com/hootsuite/atlantis/server/events/webhooks"
	"github.com/hootsuite/atlantis/server/events/webhooks/mocks"
	"github.com/hootsuite/atlantis/server/logging"
	. "github.com/hootsuite/atlantis/testing"
	. "github.com/petergtz/pegomock"
)

const (
	validEvent   = webhooks.ApplyEvent
	validRegex   = ".*"
	validKind    = webhooks.SlackKind
	validChannel = "validchannel"
)

var validConfig = webhooks.Config{
	Event:          validEvent,
	WorkspaceRegex: validRegex,
	Kind:           validKind,
	Channel:        validChannel,
}

var validConfigs = []webhooks.Config{validConfig}

func TestNewWebhooksManager_InvalidRegex(t *testing.T) {
	t.Log("When given an invalid regex in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.MockSlackClient()
	invalidRegex := "("
	configs := validConfigs
	configs[0].WorkspaceRegex = invalidRegex
	_, err := webhooks.NewWebhooksManager(configs, client)
	Assert(t, err != nil, "expected error")
	Assert(t, strings.Contains(err.Error(), "error parsing regexp"), "expected regex error")
}

func TestNewWebhooksManager_UnsupportedEvent(t *testing.T) {
	t.Log("When given an unsupported event in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.MockSlackClient()
	unsupportedEvent := "badevent"
	configs := validConfigs
	configs[0].Event = unsupportedEvent
	_, err := webhooks.NewWebhooksManager(configs, client)
	Assert(t, err != nil, "expected error")
	Equals(t, "event: badevent not supported. Only event: apply is supported right now", err.Error())
}

func TestNewWebhooksManager_UnsupportedKind(t *testing.T) {
	t.Log("When given an unsupported kind in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.MockSlackClient()
	unsupportedKind := "badkind"
	configs := validConfigs
	configs[0].Kind = unsupportedKind
	_, err := webhooks.NewWebhooksManager(configs, client)
	Assert(t, err != nil, "expected error")
	Equals(t, "kind: badkind not supported. Only kind: slack is supported right now", err.Error())
}

func TestNewWebhooksManager_NoConfigSuccess(t *testing.T) {
	t.Log("When there are no configs, everything should succeed")
	t.Log("passing any client should succeed")
	var emptyConfigs []webhooks.Config
	emptyToken := ""
	m, err := webhooks.NewWebhooksManager(emptyConfigs, webhooks.NewClient(emptyToken))
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, 0, len(m.Webhooks))

	t.Log("passing nil client hould succeed")
	m, err = webhooks.NewWebhooksManager(emptyConfigs, nil)
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, 0, len(m.Webhooks))
}
func TestNewWebhooksManager_SingleConfigSuccess(t *testing.T) {
	t.Log("One valid config should succeed")
	RegisterMockTestingT(t)
	client := mocks.MockSlackClient()
	configs := validConfigs
	m, err := webhooks.NewWebhooksManager(configs, client)
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, 1, len(m.Webhooks))
}

func TestNewWebhooksManager_MultipleConfigSuccess(t *testing.T) {
	t.Log("Multiple valid configs should succeed")
	RegisterMockTestingT(t)
	client := mocks.MockSlackClient()
	var configs []webhooks.Config
	nConfigs := 5
	for i := 0; i < nConfigs; i++ {
		configs = append(configs, validConfig)
	}
	m, err := webhooks.NewWebhooksManager(configs, client)
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, nConfigs, len(m.Webhooks))
}

func TestSend_SingleSuccess(t *testing.T) {
	t.Log("Sending one webhook should succeed")
	RegisterMockTestingT(t)
	sender := mocks.NewMockWebhookSender()
	manager := WebhooksManager{
		Webhooks: []WebhookSender{webhookSender},
	}
	logger := logging.NewNoopLogger()
	result := webhooks.ApplyResult{}
	manager.Send(logger, result)
	sender.VerifyWasCalledOnce().Send(result)
}

func TestSend_MultipleSuccess(t *testing.T) {
	t.Log("Sending multiple webhooks should succeed")
	RegisterMockTestingT(t)
	senders := []WebhookSender{
		mocks.NewMockWebhookSender(),
		mocks.NewMockWebhookSender(),
		mocks.NewMockWebhookSender(),
	}
	manager := WebhooksManager{
		Webhooks: senders,
	}
	logger := logging.NewNoopLogger()
	result := webhooks.ApplyResult{}
	manager.Send(logger, result)
	for _, s := range senders {
		s.VerifyWasCalledOnce().Send(result)
	}
}
