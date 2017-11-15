package webhooks

import (
	"fmt"
	"regexp"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const (
	successColour = "good"
	failureColour = "danger"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_slack.go SlackClient

type SlackClient interface {
	AuthTest() error
	ChannelExist(channelName string) (bool, error)
	PostMessage(channel string, applyResult ApplyResult) error
}

type ConcreteSlackClient struct {
	Slack *slack.Client
}

type SlackWebhook struct {
	Client   SlackClient
	EnvRegex *regexp.Regexp
	Channel  string
}

func NewSlackClient(token string) SlackClient {
	return &ConcreteSlackClient{
		Slack: slack.New(token),
	}
}

func NewSlack(r *regexp.Regexp, channel string, client SlackClient) (*SlackWebhook, error) {
	if err := client.AuthTest(); err != nil {
		return nil, errors.Wrap(err, "testing slack authentication")
	}

	channelExist, err := client.ChannelExist(channel)
	if err != nil {
		return nil, err
	}
	if !channelExist {
		return nil, errors.Errorf("slack channel %q doesn't exist", channel)
	}

	return &SlackWebhook{
		Client:   client,
		EnvRegex: r,
		Channel:  channel,
	}, nil
}

func (c *ConcreteSlackClient) AuthTest() error {
	_, err := c.Slack.AuthTest()
	return err
}

func (c *ConcreteSlackClient) ChannelExist(channelName string) (bool, error) {
	channels, err := c.Slack.GetChannels(true)
	if err != nil {
		return false, err
	}

	for _, channel := range channels {
		if channel.Name == channelName {
			return true, nil
		}
	}
	return false, nil
}

func (c *ConcreteSlackClient) PostMessage(channel string, applyResult ApplyResult) error {
	params := slack.NewPostMessageParameters()
	params.Attachments = c.createAttachments(applyResult)
	params.AsUser = true
	params.EscapeText = false
	_, _, err := c.Slack.PostMessage(channel, "", params)
	return err
}

func (c *ConcreteSlackClient) createAttachments(applyResult ApplyResult) []slack.Attachment {
	var colour string
	if applyResult.Success {
		colour = successColour
	} else {
		colour = failureColour
	}

	text := fmt.Sprintf("Applied in <%s|%s>.", applyResult.Pull.URL, applyResult.Repo.FullName)
	attachment := slack.Attachment{
		Color: colour,
		Text:  text,
		Fields: []slack.AttachmentField{
			slack.AttachmentField{
				Title: "Environment",
				Value: applyResult.Environment,
				Short: true,
			},
			slack.AttachmentField{
				Title: "User",
				Value: applyResult.User.Username,
				Short: true,
			},
		},
	}
	return []slack.Attachment{attachment}
}

func (s *SlackWebhook) Send(applyResult ApplyResult) error {
	if !s.EnvRegex.MatchString(applyResult.Environment) {
		return nil
	}
	return s.Client.PostMessage(s.Channel, applyResult)
}
