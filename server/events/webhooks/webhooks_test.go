package webhooks_test

import (
	"strings"
	"testing"

	"github.com/hootsuite/atlantis/server/events/webhooks"
	. "github.com/hootsuite/atlantis/testing"
)

const (
	validToken   = "validtoken"
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
	var configs []webhooks.Config
	return append(configs, validConfig)
}
func TestNewWebhooksManager_InvalidRegex(t *testing.T) {
	invalidRegex := "("
	configs := validConfigs()
	configs[0].WorkspaceRegex = invalidRegex
	_, err := webhooks.NewWebhooksManager(configs, validToken)
	Assert(t, strings.Contains(err.Error(), "error parsing regexp"), "expected err")
}

func TestNewWebhooksManager_UnsupportedEvent(t *testing.T) {
	unsupportedEvent := "badevent"
	configs := validConfigs()
	configs[0].Event = unsupportedEvent
	_, err := webhooks.NewWebhooksManager(configs, validToken)
	Assert(t, strings.Contains(err.Error(), "event"), "expected err")
}

func TestNewWebhooksManager_UnsupportedKind(t *testing.T) {
	unsupportedKind := "badkind"
	configs := validConfigs()
	configs[0].Kind = unsupportedKind
	_, err := webhooks.NewWebhooksManager(configs, validToken)
	Assert(t, strings.Contains(err.Error(), "kind"), "expected err")
}

func TestNewWebhooksManager_NoSlackToken(t *testing.T) {
	emptyToken := ""
	_, err := webhooks.NewWebhooksManager(validConfigs(), emptyToken)
	Assert(t, strings.Contains(err.Error(), "slack-token must be set"), "expected err")
}

func TestNewWebhooksManager_SingleSuccess(t *testing.T) {
	// todo: failing because not a valid token and channel
	m, err := webhooks.NewWebhooksManager(validConfigs(), validToken)
	Ok(t, err)
	Assert(t, m != nil, "mangager shouldn't be nil")
}

func TestNewWebhooksManager_MultipleSuccess(t *testing.T) {
	// todo: failing because not a valid token and channel
	configs := validConfigs()
	for i := 0; i < 5; i++ {
		configs = append(configs, validConfig)
	}
	m, err := webhooks.NewWebhooksManager(configs, validToken)
	Ok(t, err)
	Assert(t, m != nil, "mangager shouldn't be nil")
}

func TestSend_Err(t *testing.T) {
	// todo: test
}

func TestSend_Success(t *testing.T) {
	// todo: test
}
