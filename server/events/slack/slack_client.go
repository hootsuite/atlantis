package slack

import (
	"errors"

	"github.com/nlopes/slack"
)

type Client interface {
	PostMessage(text string) (string, error)
}

type ConcreteClient struct {
	client  *slack.Client
	channel string
}

func NewClient(slackToken string, channelName string) (*ConcreteClient, error) {
	slackClient := slack.New(slackToken)

	if _, err := slackClient.AuthTest(); err != nil {
		return nil, err
	}

	// https://api.slack.com/faq
	// 'How do I find a channel's ID if I only have its #name?'
	// says need to look through all channels and match the name
	channels, err := slackClient.GetChannels(true)
	if err != nil {
		return nil, err
	}
	for _, c := range channels {
		if c.Name == channelName {
			// channel exists, no errors
			return &ConcreteClient{
				client:  slackClient,
				channel: channelName,
			}, nil
		}
	}

	return nil, errors.New("channel_not_found")
}

func (s *ConcreteClient) PostMessage(text string) (string, error) {
	params := slack.NewPostMessageParameters()
	params.AsUser = true
	params.EscapeText = false

	_, timestamp, err := s.client.PostMessage(s.channel, text, params)
	return timestamp, err
}
