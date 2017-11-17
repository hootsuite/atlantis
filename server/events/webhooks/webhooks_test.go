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

func validConfigs() []webhooks.Config {
	return []webhooks.Config{validConfig}
}

func TestNewWebhooksManager_InvalidRegex(t *testing.T) {
	t.Log("When given an invalid regex in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	When(client.ChannelExists(validChannel)).ThenReturn(true, nil)

	invalidRegex := "("
	configs := validConfigs()
	configs[0].WorkspaceRegex = invalidRegex
	_, err := webhooks.NewMultiWebhookSender(configs, client)
	Assert(t, err != nil, "expected error")
	Assert(t, strings.Contains(err.Error(), "error parsing regexp"), "expected regex error")
}

func TestNewWebhooksManager_NoEvent(t *testing.T) {
	t.Log("When the event key is not specified in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	configs := validConfigs()
	configs[0].Event = ""
	_, err := webhooks.NewMultiWebhookSender(configs, client)
	Assert(t, err != nil, "expected error")
	Equals(t, "must specify \"kind\" and \"event\" keys for webhooks", err.Error())
}

func TestNewWebhooksManager_UnsupportedEvent(t *testing.T) {
	t.Log("When given an unsupported event in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	When(client.ChannelExists(validChannel)).ThenReturn(true, nil)

	unsupportedEvent := "badevent"
	configs := validConfigs()
	configs[0].Event = unsupportedEvent
	_, err := webhooks.NewMultiWebhookSender(configs, client)
	Assert(t, err != nil, "expected error")
	Equals(t, "\"event: badevent\" not supported. Only \"event: apply\" is supported right now", err.Error())
}

func TestNewWebhooksManager_NoKind(t *testing.T) {
	t.Log("When the kind key is not specified in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	configs := validConfigs()
	configs[0].Kind = ""
	_, err := webhooks.NewMultiWebhookSender(configs, client)
	Assert(t, err != nil, "expected error")
	Equals(t, "must specify \"kind\" and \"event\" keys for webhooks", err.Error())
}

func TestNewWebhooksManager_UnsupportedKind(t *testing.T) {
	t.Log("When given an unsupported kind in a config, an error is returned")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	When(client.ChannelExists(validChannel)).ThenReturn(true, nil)

	unsupportedKind := "badkind"
	configs := validConfigs()
	configs[0].Kind = unsupportedKind
	_, err := webhooks.NewMultiWebhookSender(configs, client)
	Assert(t, err != nil, "expected error")
	Equals(t, "\"kind: badkind\" not supported. Only \"kind: slack\" is supported right now", err.Error())
}

func TestNewWebhooksManager_NoConfigSuccess(t *testing.T) {
	t.Log("When there are no configs, function should succeed")
	t.Log("passing any client should succeed")
	var emptyConfigs []webhooks.Config
	emptyToken := ""
	m, err := webhooks.NewMultiWebhookSender(emptyConfigs, webhooks.NewSlackClient(emptyToken))
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, 0, len(m.Webhooks))

	t.Log("passing nil client hould succeed")
	m, err = webhooks.NewMultiWebhookSender(emptyConfigs, nil)
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, 0, len(m.Webhooks))
}
func TestNewWebhooksManager_SingleConfigSuccess(t *testing.T) {
	t.Log("When there is one valid config, function should succeed")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	When(client.TokenIsSet()).ThenReturn(true)
	When(client.ChannelExists(validChannel)).ThenReturn(true, nil)

	configs := validConfigs()
	m, err := webhooks.NewMultiWebhookSender(configs, client)
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, 1, len(m.Webhooks))
}

func TestNewWebhooksManager_MultipleConfigSuccess(t *testing.T) {
	t.Log("When there are multiple valid configs, function should succeed")
	RegisterMockTestingT(t)
	client := mocks.NewMockSlackClient()
	When(client.TokenIsSet()).ThenReturn(true)
	When(client.ChannelExists(validChannel)).ThenReturn(true, nil)

	var configs []webhooks.Config
	nConfigs := 5
	for i := 0; i < nConfigs; i++ {
		configs = append(configs, validConfig)
	}
	m, err := webhooks.NewMultiWebhookSender(configs, client)
	Ok(t, err)
	Assert(t, m != nil, "manager shouldn't be nil")
	Equals(t, nConfigs, len(m.Webhooks))
}

func TestSend_SingleSuccess(t *testing.T) {
	t.Log("Sending one webhook should succeed")
	RegisterMockTestingT(t)
	sender := mocks.NewMockSender()
	manager := webhooks.MultiWebhookSender{
		Webhooks: []webhooks.Sender{sender},
	}
	logger := logging.NewNoopLogger()
	result := webhooks.ApplyResult{}
	manager.Send(logger, result) // nolint: errcheck
	sender.VerifyWasCalledOnce().Send(logger, result)
}

func TestSend_MultipleSuccess(t *testing.T) {
	t.Log("Sending multiple webhooks should succeed")
	RegisterMockTestingT(t)
	senders := []*mocks.MockSender{
		mocks.NewMockSender(),
		mocks.NewMockSender(),
		mocks.NewMockSender(),
	}
	manager := webhooks.MultiWebhookSender{
		Webhooks: []webhooks.Sender{senders[0], senders[1], senders[2]},
	}
	logger := logging.NewNoopLogger()
	result := webhooks.ApplyResult{}
	err := manager.Send(logger, result)
	Ok(t, err)
	for _, s := range senders {
		s.VerifyWasCalledOnce().Send(logger, result)
	}
}
