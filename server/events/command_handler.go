package events

import (
	"fmt"

	"github.com/google/go-github/github"
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/events/vcs"
	"github.com/hootsuite/atlantis/server/logging"
	"github.com/hootsuite/atlantis/server/recovery"
	"github.com/lkysow/go-gitlab"
	"github.com/pkg/errors"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_command_runner.go CommandRunner

type CommandRunner interface {
	ExecuteCommand(baseRepo models.Repo, headRepo models.Repo, user models.User, pullNum int, cmd *Command, vcsHost vcs.Host)
}

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_github_pull_getter.go GithubPullGetter

type GithubPullGetter interface {
	GetPullRequest(repo models.Repo, pullNum int) (*github.PullRequest, error)
}

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_gitlab_merge_request_getter.go GitlabMergeRequestGetter

type GitlabMergeRequestGetter interface {
	GetMergeRequest(repoFullName string, pullNum int) (*gitlab.MergeRequest, error)
}

// CommandHandler is the first step when processing a comment command.
type CommandHandler struct {
	PlanExecutor             Executor
	ApplyExecutor            Executor
	HelpExecutor             Executor
	LockURLGenerator         LockURLGenerator
	VCSClient                vcs.ClientProxy
	GithubPullGetter         GithubPullGetter
	GitlabMergeRequestGetter GitlabMergeRequestGetter
	GHStatus                 GHStatusUpdater
	EventParser              EventParsing
	EnvLocker                EnvLocker
	GHCommentRenderer        *GithubCommentRenderer
	Logger                   logging.SimpleLogging
}

// ExecuteCommand executes the command
func (c *CommandHandler) ExecuteCommand(baseRepo models.Repo, headRepo models.Repo, user models.User, pullNum int, cmd *Command, vcsHost vcs.Host) {
	var err error
	var pull models.PullRequest
	if vcsHost == vcs.Github {
		pull, headRepo, err = c.getGithubData(baseRepo, pullNum)
	} else if vcsHost == vcs.Gitlab {
		pull, err = c.getGitlabData(baseRepo.FullName, pullNum)
	}

	log := c.buildLogger(baseRepo.FullName, pullNum)
	if err != nil {
		log.Err(err.Error())
		return
	}
	ctx := &CommandContext{
		User:     user,
		Log:      log,
		Pull:     pull,
		HeadRepo: headRepo,
		Command:  cmd,
		VCSHost:  vcsHost,
		BaseRepo: baseRepo,
	}
	c.run(ctx)
}

func (c *CommandHandler) getGithubData(baseRepo models.Repo, pullNum int) (models.PullRequest, models.Repo, error) {
	if c.GithubPullGetter == nil {
		return models.PullRequest{}, models.Repo{}, errors.New("Atlantis not configured to support GitHub")
	}
	ghPull, err := c.GithubPullGetter.GetPullRequest(baseRepo, pullNum)
	if err != nil {
		return models.PullRequest{}, models.Repo{}, errors.Wrap(err, "making pull request API call to GitHub")
	}
	pull, repo, err := c.EventParser.ParseGithubPull(ghPull)
	if err != nil {
		return pull, repo, errors.Wrap(err, "extracting required fields from comment data")
	}
	return pull, repo, nil
}

func (c *CommandHandler) getGitlabData(repoFullName string, pullNum int) (models.PullRequest, error) {
	if c.GitlabMergeRequestGetter == nil {
		return models.PullRequest{}, errors.New("Atlantis not configured to support GitLab")
	}
	mr, err := c.GitlabMergeRequestGetter.GetMergeRequest(repoFullName, pullNum)
	if err != nil {
		return models.PullRequest{}, errors.Wrap(err, "making merge request API call to GitLab")
	}
	pull := c.EventParser.ParseGitlabMergeRequest(mr)
	return pull, nil
}

func (c *CommandHandler) buildLogger(repoFullName string, pullNum int) *logging.SimpleLogger {
	src := fmt.Sprintf("%s#%d", repoFullName, pullNum)
	return logging.NewSimpleLogger(src, c.Logger.Underlying(), true, c.Logger.GetLevel())
}

func (c *CommandHandler) SetLockURL(f func(id string) (url string)) {
	c.LockURLGenerator.SetLockURL(f)
}
func (c *CommandHandler) run(ctx *CommandContext) {
	log := c.buildLogger(ctx.BaseRepo.FullName, ctx.Pull.Num)
	ctx.Log = log
	defer c.logPanics(ctx)

	if ctx.Pull.State != models.Open {
		ctx.Log.Info("command was run on closed pull request")
		c.VCSClient.CreateComment(ctx.BaseRepo, ctx.Pull, "Atlantis commands can't be run on closed pull requests", ctx.VCSHost) // nolint: errcheck
		return
	}

	c.GHStatus.Update(ctx.BaseRepo, ctx.Pull, vcs.Pending, ctx.Command, ctx.VCSHost) // nolint: errcheck
	if !c.EnvLocker.TryLock(ctx.BaseRepo.FullName, ctx.Command.Environment, ctx.Pull.Num) {
		errMsg := fmt.Sprintf(
			"The %s environment is currently locked by another"+
				" command that is running for this pull request."+
				" Wait until the previous command is complete and try again.",
			ctx.Command.Environment)
		ctx.Log.Warn(errMsg)
		c.updatePull(ctx, CommandResponse{Failure: errMsg})
		return
	}
	defer c.EnvLocker.Unlock(ctx.BaseRepo.FullName, ctx.Command.Environment, ctx.Pull.Num)

	var cr CommandResponse
	switch ctx.Command.Name {
	case Plan:
		cr = c.PlanExecutor.Execute(ctx)
	case Apply:
		cr = c.ApplyExecutor.Execute(ctx)
	case Help:
		cr = c.HelpExecutor.Execute(ctx)
	default:
		ctx.Log.Err("failed to determine desired command, neither plan nor apply")
	}
	c.updatePull(ctx, cr)
}

func (c *CommandHandler) updatePull(ctx *CommandContext, res CommandResponse) {
	// Log if we got any errors or failures.
	if res.Error != nil {
		ctx.Log.Err(res.Error.Error())
	} else if res.Failure != "" {
		ctx.Log.Warn(res.Failure)
	}

	// Update the pull request's status icon and comment back.
	c.GHStatus.UpdateProjectResult(ctx, res) // nolint: errcheck
	comment := c.GHCommentRenderer.Render(res, ctx.Command.Name, ctx.Log.History.String(), ctx.Command.Verbose)
	c.VCSClient.CreateComment(ctx.BaseRepo, ctx.Pull, comment, ctx.VCSHost) // nolint: errcheck
}

// logPanics logs and creates a comment on the pull request for panics
func (c *CommandHandler) logPanics(ctx *CommandContext) {
	if err := recover(); err != nil {
		stack := recovery.Stack(3)
		c.VCSClient.CreateComment(ctx.BaseRepo, ctx.Pull, // nolint: errcheck
			fmt.Sprintf("**Error: goroutine panic. This is a bug.**\n```\n%s\n%s```", err, stack), ctx.VCSHost)
		ctx.Log.Err("PANIC: %s\n%s", err, stack)
	}
}
