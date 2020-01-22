package webhooks

import (
	"fmt"

	"github.com/nlopes/slack"
)

const (
	slackSuccessColour = "good"
	slackFailureColour = "danger"
)

//go:generate pegomock generate -m --use-experimental-model-gen --package mocks -o mocks/mock_slack_client.go SlackClient

// SlackClient handles making API calls to Slack.
type SlackClient interface {
	AuthTest() error
	TokenIsSet() bool
	ChannelExists(channelName string) (bool, error)
	PostMessage(channel string, applyResult ApplyResult) error
}

//go:generate pegomock generate -m --use-experimental-model-gen --package mocks -o mocks/mock_underlying_slack_client.go UnderlyingSlackClient

// UnderlyingSlackClient wraps the nlopes/slack.Client implementation so
// we can mock it during tests.
type UnderlyingSlackClient interface {
	AuthTest() (response *slack.AuthTestResponse, error error)
	GetChannels(excludeArchived bool) ([]slack.Channel, error)
	PostMessage(channel, text string, parameters slack.PostMessageParameters) (string, string, error)
}

type DefaultSlackClient struct {
	Slack UnderlyingSlackClient
	Token string
}

func NewSlackClient(token string) SlackClient {
	return &DefaultSlackClient{
		Slack: slack.New(token),
		Token: token,
	}
}

func (d *DefaultSlackClient) AuthTest() error {
	_, err := d.Slack.AuthTest()
	return err
}

func (d *DefaultSlackClient) TokenIsSet() bool {
	return d.Token != ""
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
	var successWord string
	if applyResult.Success {
		colour = slackSuccessColour
		successWord = "succeeded"
	} else {
		colour = slackFailureColour
		successWord = "failed"
	}

	text := fmt.Sprintf("Apply %s for <%s|%s>", successWord, applyResult.Pull.URL, applyResult.Repo.FullName)
	attachment := slack.Attachment{
		Color: colour,
		Text:  text,
		Fields: []slack.AttachmentField{
			{
				Title: "Workspace",
				Value: applyResult.Workspace,
				Short: true,
			},
			{
				Title: "User",
				Value: applyResult.User.Username,
				Short: true,
			},
		},
	}
	return []slack.Attachment{attachment}
}

