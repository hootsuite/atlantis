package events

import (
	"github.com/hootsuite/atlantis/server/events/slack"
)

type HookExecutor interface {
	ExecuteHook(ctx *CommandContext) error
}

type SlackhookExecutor struct {
	Slack   slack.Client
	Channel string
}

func (s *SlackhookExecutor) ExecuteHook(ctx *CommandContext) error {
	// TODO: think about passing down success/fail result into ExecuteHook?
	template := slack.SlackMessageTemplate{
		Success:     true,
		Username:    ctx.User.Username,
		CommandName: ctx.Command.Name.String() + " " + ctx.Command.Environment,
		RepoName:    ctx.BaseRepo.Name,
		PullURL:     ctx.Pull.URL,
	}

	_, err := s.Slack.PostMessage(s.Channel, template)
	return err
}
