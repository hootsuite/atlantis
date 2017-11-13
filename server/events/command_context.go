package events

import (
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/logging"
	"github.com/hootsuite/atlantis/server/vcs"
)

type CommandContext struct {
	BaseRepo models.Repo
	HeadRepo models.Repo
	Pull     models.PullRequest
	User     models.User
	Command  *Command
	Log      *logging.SimpleLogger
	VCSHost  vcs.Host
}
