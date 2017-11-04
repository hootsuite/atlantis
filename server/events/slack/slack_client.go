package slack

import (
	"fmt"

	"github.com/nlopes/slack"
)

type Client interface {
	PostMessage(channel string, template SlackMessageTemplate) (string, error)
}

type ConcreteClient struct {
	client *slack.Client
}

type SlackMessageTemplate struct {
	Success     bool
	Username    string
	CommandName string
	RepoName    string
	PullURL     string
}

func NewClient(slackToken string) (Client, error) {
	slackClient := slack.New(slackToken)

	_, err := slackClient.AuthTest()
	if err != nil {
		return nil, err
	}

	return &ConcreteClient{
		client: slackClient,
	}, nil
}

func (s *ConcreteClient) PostMessage(channel string, template SlackMessageTemplate) (string, error) {
	params := slack.NewPostMessageParameters()
	params.AsUser = true
	params.EscapeText = false

	text := s.createMessage(template)
	_, timestamp, err := s.client.PostMessage(channel, text, params)
	return timestamp, err
}

func (s *ConcreteClient) createMessage(template SlackMessageTemplate) string {
	var status string
	if template.Success {
		status = ":white_check_mark:"
	} else {
		status = ":x:"
	}
	return fmt.Sprintf("%s *%s* %s in <%s|%s>.",
		status,
		template.Username,
		template.CommandName,
		template.PullURL,
		template.RepoName)
}
