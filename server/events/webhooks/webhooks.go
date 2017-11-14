package webhooks

import (
	"fmt"
	"regexp"

	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/logging"
)

const SlackKind = "slack"
const ApplyEvent = "apply"

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_webhooks.go webhooks.go

type WebhooksSender interface {
	Send(log *logging.SimpleLogger, result ApplyResult)
}

type WebhookSender interface {
	Send(ApplyResult) error
}

type ApplyResult struct {
	Environment string
	Repo        models.Repo
	Pull        models.PullRequest
	User        models.User
	Success     bool
}

type WebhooksManager struct {
	Webhooks []WebhookSender
}

type Config struct {
	Event          string
	WorkspaceRegex string
	Kind           string
	Channel        string
}

func NewWebhooksManager(configs []Config, client SlackClient) (*WebhooksManager, error) {
	var webhooks []WebhookSender
	for _, c := range configs {
		r, err := regexp.Compile(c.WorkspaceRegex)
		if err != nil {
			return nil, err
		}
		if c.Event != ApplyEvent {
			return nil, fmt.Errorf("event: %s not supported. Only event: %s is supported right now", c.Event, ApplyEvent)
		}
		switch c.Kind {
		case SlackKind:
			slack, err := NewSlack(r, c.Channel, client)
			if err != nil {
				return nil, err
			}
			webhooks = append(webhooks, slack)
		default:
			return nil, fmt.Errorf("kind: %s not supported. Only kind: %s is supported right now", c.Kind, SlackKind)
		}
	}

	return &WebhooksManager{
		Webhooks: webhooks,
	}, nil
}

func (w *WebhooksManager) Send(log *logging.SimpleLogger, result ApplyResult) {
	for _, w := range w.Webhooks {
		if err := w.Send(result); err != nil {
			log.Warn("error sending slack webhook: %s", err)
		}
	}
}
