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
	params.Attachments = s.createAttachments(result)
	params.AsUser = true
	params.EscapeText = false
	_, _, err := s.Client.PostMessage(s.Channel, "", params)
	return err
}

func (s *SlackWebhook) createAttachments(result ApplyResult) []slack.Attachment {
	var color string
	if result.Success {
		color = "good"
	} else {
		color = "danger"
	}

	text := fmt.Sprintf("Applied in <%s|%s>.", result.Pull.URL, result.Repo.FullName)
	attachment := slack.Attachment{
		Color: color,
		Text:  text,
		Fields: []slack.AttachmentField{
			slack.AttachmentField{
				Title: "Environment",
				Value: result.Environment,
				Short: true,
			},
			slack.AttachmentField{
				Title: "User",
				Value: result.User.Username,
				Short: true,
			},
		},
	}
	var attachments []slack.Attachment
	return append(attachments, attachment)
}
