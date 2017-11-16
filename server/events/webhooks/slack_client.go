package webhooks

import (
	"fmt"

	"github.com/nlopes/slack"
)

const (
	slackSuccessColour = "good"
	slackFailureColour = "danger"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_slack_client.go SlackClient

type SlackClient interface {
	AuthTest() error
	ChannelExists(channelName string) (bool, error)
	PostMessage(channel string, applyResult ApplyResult) error
}

type DefaultSlackClient struct {
	Slack *slack.Client
}

func NewSlackClient(token string) SlackClient {
	return &DefaultSlackClient{
		Slack: slack.New(token),
	}
}

func (d *DefaultSlackClient) AuthTest() error {
	_, err := d.Slack.AuthTest()
	return err
}

func (d *DefaultSlackClient) ChannelExists(channelName string) (bool, error) {
	channels, err := d.Slack.GetChannels(true)
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

func (d *DefaultSlackClient) PostMessage(channel string, applyResult ApplyResult) error {
	params := slack.NewPostMessageParameters()
	params.Attachments = d.createAttachments(applyResult)
	params.AsUser = true
	params.EscapeText = false
	_, _, err := d.Slack.PostMessage(channel, "", params)
	return err
}

func (d *DefaultSlackClient) createAttachments(applyResult ApplyResult) []slack.Attachment {
	var colour string
	if applyResult.Success {
		colour = slackSuccessColour
	} else {
		colour = slackFailureColour
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
