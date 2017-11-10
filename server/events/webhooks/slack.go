package webhooks

import (
	"fmt"
	"regexp"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

type SlackWebhook struct {
	EnvRegex *regexp.Regexp
	Channel  string
	Token    string
	Client   *slack.Client
}

func NewSlack(r *regexp.Regexp, channel string, token string) (*SlackWebhook, error) {
	slackClient := slack.New(token)
	if _, err := slackClient.AuthTest(); err != nil {
		return nil, errors.Wrap(err, "testing slack authentication")
	}

	// Make sure the slack channel exists.
	channels, err := slackClient.GetChannels(true)
	if err != nil {
		return nil, err
	}
	channelExist := false
	for _, c := range channels {
		if c.Name == channel {
			channelExist = true
			break
		}
	}
	if !channelExist {
		return nil, errors.Errorf("slack channel %q doesn't exist", channel)
	}

	return &SlackWebhook{
		Client:   slackClient,
		EnvRegex: r,
		Channel:  channel,
		Token:    token,
	}, nil
}

func (s *SlackWebhook) Send(result ApplyResult) error {
	if !s.EnvRegex.MatchString(result.Environment) {
		return nil
	}

	params := slack.NewPostMessageParameters()
	params.AsUser = true
	params.EscapeText = false
	text := s.createMessage(result)
	_, _, err := s.Client.PostMessage(s.Channel, text, params)
	return err
}

func (s *SlackWebhook) createMessage(result ApplyResult) string {
	var status string
	if result.Success {
		status = ":white_check_mark:"
	} else {
		status = ":x:"
	}
	return fmt.Sprintf("%s *%s* %s in <%s|%s>.",
		status,
		result.User.Username,
		"apply",
		result.Pull.URL,
		result.Repo.FullName)
}
