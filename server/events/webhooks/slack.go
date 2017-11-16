package webhooks

import (
	"regexp"

	"github.com/pkg/errors"
)

type SlackWebhook struct {
	Client   SlackClient
	EnvRegex *regexp.Regexp
	Channel  string
}

func NewSlack(r *regexp.Regexp, channel string, client SlackClient) (*SlackWebhook, error) {
	if err := client.AuthTest(); err != nil {
		return nil, errors.Wrap(err, "testing slack authentication")
	}

	channelExists, err := client.ChannelExists(channel)
	if err != nil {
		return nil, err
	}
	if !channelExists {
		return nil, errors.Errorf("slack channel %q doesn't exist", channel)
	}

	return &SlackWebhook{
		Client:   client,
		EnvRegex: r,
		Channel:  channel,
	}, nil
}

func (s *SlackWebhook) Send(applyResult ApplyResult) error {
	if !s.EnvRegex.MatchString(applyResult.Environment) {
		return nil
	}
	return s.Client.PostMessage(s.Channel, applyResult)
}
