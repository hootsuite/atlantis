package events

import (
	"fmt"
	"strings"

	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/events/vcs"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_commit_status_updater.go CommitStatusUpdater

type CommitStatusUpdater interface {
	Update(repo models.Repo, pull models.PullRequest, status vcs.CommitStatus, cmd *Command, host vcs.Host) error
	UpdateProjectResult(ctx *CommandContext, res CommandResponse) error
}

type DefaultCommitStatusUpdater struct {
	Client vcs.ClientProxy
}

func (g *DefaultCommitStatusUpdater) Update(repo models.Repo, pull models.PullRequest, status vcs.CommitStatus, cmd *Command, host vcs.Host) error {
	description := fmt.Sprintf("%s %s", strings.Title(cmd.Name.String()), strings.Title(status.String()))
	return g.Client.UpdateStatus(repo, pull, status, description, host)
}

func (g *DefaultCommitStatusUpdater) UpdateProjectResult(ctx *CommandContext, res CommandResponse) error {
	var status vcs.CommitStatus
	if res.Error != nil || res.Failure != "" {
		status = vcs.Failed
	} else {
		var statuses []vcs.CommitStatus
		for _, p := range res.ProjectResults {
			statuses = append(statuses, p.Status())
		}
		status = g.worstStatus(statuses)
	}
	return g.Update(ctx.BaseRepo, ctx.Pull, status, ctx.Command, ctx.VCSHost)
}

func (g *DefaultCommitStatusUpdater) worstStatus(ss []vcs.CommitStatus) vcs.CommitStatus {
	for _, s := range ss {
		if s == vcs.Failed {
			return vcs.Failed
		}
	}
	return vcs.Success
}
