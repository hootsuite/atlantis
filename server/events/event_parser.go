package events

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/vcs"
	"github.com/xanzy/go-gitlab"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_event_parsing.go EventParsing

type Command struct {
	Name        CommandName
	Environment string
	Verbose     bool
	Flags       []string
}

type EventParsing interface {
	DetermineCommand(comment string, vcsHost vcs.Host) (*Command, error)
	ParseGithubIssueCommentEvent(comment *github.IssueCommentEvent) (baseRepo models.Repo, user models.User, pullNum int, err error)
	ParseGithubPull(pull *github.PullRequest) (models.PullRequest, models.Repo, error)
	ParseGithubRepo(ghRepo *github.Repository) (models.Repo, error)
	ParseGitlabMergeEvent(event gitlab.MergeEvent) (models.PullRequest, models.Repo)
	ParseGitlabMergeCommentEvent(event gitlab.MergeCommentEvent) (baseRepo models.Repo, headRepo models.Repo, user models.User)
	ParseGitlabMergeRequest(mr *gitlab.MergeRequest) models.PullRequest
}

type EventParser struct {
	GithubUser  string
	GithubToken string
	GitlabUser  string
	GitlabToken string
}

// DetermineCommand parses the comment as an atlantis command. If it succeeds,
// it returns the command. Otherwise it returns error.
func (e *EventParser) DetermineCommand(comment string, vcsHost vcs.Host) (*Command, error) {
	// valid commands contain:
	// the initial "executable" name, 'run' or 'atlantis' or '@GithubUser' where GithubUser is the api user atlantis is running as
	// then a command, either 'plan', 'apply', or 'help'
	// then an optional environment argument, an optional --verbose flag and any other flags
	//
	// examples:
	// atlantis help
	// run plan
	// @GithubUser plan staging
	// atlantis plan staging --verbose
	// atlantis plan staging --verbose -key=value -key2 value2
	err := errors.New("not an Atlantis command")
	args := strings.Fields(comment)
	if len(args) < 2 {
		return nil, err
	}

	env := "default"
	verbose := false
	var flags []string

	vcsUser := e.GithubUser
	if vcsHost == vcs.Gitlab {
		vcsUser = e.GitlabUser
	}
	if !e.stringInSlice(args[0], []string{"run", "atlantis", "@" + vcsUser}) {
		return nil, err
	}
	if !e.stringInSlice(args[1], []string{"plan", "apply", "help"}) {
		return nil, err
	}
	if args[1] == "help" {
		return &Command{Name: Help}, nil
	}
	command := args[1]

	if len(args) > 2 {
		flags = args[2:]

		// if the third arg doesn't start with '-' then we assume it's an
		// environment not a flag
		if !strings.HasPrefix(args[2], "-") {
			env = args[2]
			flags = args[3:]
		}

		// check for --verbose specially and then remove any additional
		// occurrences
		if e.stringInSlice("--verbose", flags) {
			verbose = true
			flags = e.removeOccurrences("--verbose", flags)
		}
	}

	c := &Command{Verbose: verbose, Environment: env, Flags: flags}
	switch command {
	case "plan":
		c.Name = Plan
	case "apply":
		c.Name = Apply
	default:
		return nil, fmt.Errorf("something went wrong parsing the command, the command we parsed %q was not apply or plan", command)
	}
	return c, nil
}

func (e *EventParser) ParseGithubIssueCommentEvent(comment *github.IssueCommentEvent) (baseRepo models.Repo, user models.User, pullNum int, err error) {
	baseRepo, err = e.ParseGithubRepo(comment.Repo)
	if err != nil {
		return
	}
	pullCreator := comment.Issue.User.GetLogin()
	if pullCreator == "" {
		err = errors.New("issue.user.login is null")
		return
	}
	htmlURL := comment.Issue.GetHTMLURL()
	if htmlURL == "" {
		err = errors.New("issue.html_url is null")
		return
	}
	commentorUsername := comment.Comment.User.GetLogin()
	if commentorUsername == "" {
		err = errors.New("comment.user.login is null")
		return
	}
	user = models.User{
		Username: commentorUsername,
	}
	pullNum = comment.Issue.GetNumber()
	if pullNum == 0 {
		err = errors.New("issue.number is null")
		return
	}
	return
}

func (e *EventParser) ParseGithubPull(pull *github.PullRequest) (models.PullRequest, models.Repo, error) {
	var pullModel models.PullRequest
	var headRepoModel models.Repo

	commit := pull.Head.GetSHA()
	if commit == "" {
		return pullModel, headRepoModel, errors.New("head.sha is null")
	}
	url := pull.GetHTMLURL()
	if url == "" {
		return pullModel, headRepoModel, errors.New("html_url is null")
	}
	branch := pull.Head.GetRef()
	if branch == "" {
		return pullModel, headRepoModel, errors.New("head.ref is null")
	}
	authorUsername := pull.User.GetLogin()
	if authorUsername == "" {
		return pullModel, headRepoModel, errors.New("user.login is null")
	}
	num := pull.GetNumber()
	if num == 0 {
		return pullModel, headRepoModel, errors.New("number is null")
	}

	headRepoModel, err := e.ParseGithubRepo(pull.Head.Repo)
	if err != nil {
		return pullModel, headRepoModel, err
	}

	pullState := models.Closed
	if pull.GetState() == "open" {
		pullState = models.Open
	}

	return models.PullRequest{
		Author:     authorUsername,
		Branch:     branch,
		HeadCommit: commit,
		URL:        url,
		Num:        num,
		State:      pullState,
	}, headRepoModel, nil
}

func (e *EventParser) ParseGithubRepo(ghRepo *github.Repository) (models.Repo, error) {
	var repo models.Repo
	repoFullName := ghRepo.GetFullName()
	if repoFullName == "" {
		return repo, errors.New("repository.full_name is null")
	}
	repoOwner := ghRepo.Owner.GetLogin()
	if repoOwner == "" {
		return repo, errors.New("repository.owner.login is null")
	}
	repoName := ghRepo.GetName()
	if repoName == "" {
		return repo, errors.New("repository.name is null")
	}
	repoSanitizedCloneURL := ghRepo.GetCloneURL()
	if repoSanitizedCloneURL == "" {
		return repo, errors.New("repository.clone_url is null")
	}

	// Construct HTTPS repo clone url string with username and password.
	repoCloneURL := strings.Replace(repoSanitizedCloneURL, "https://", fmt.Sprintf("https://%s:%s@", e.GithubUser, e.GithubToken), -1)

	return models.Repo{
		Owner:             repoOwner,
		FullName:          repoFullName,
		CloneURL:          repoCloneURL,
		SanitizedCloneURL: repoSanitizedCloneURL,
		Name:              repoName,
	}, nil
}

func (e *EventParser) ParseGitlabMergeEvent(event gitlab.MergeEvent) (models.PullRequest, models.Repo) {
	modelState := models.Closed
	if event.ObjectAttributes.State == "opened" {
		modelState = models.Open
	}
	// GitLab also has a "merged" state, but we map that to Closed so we don't
	// need to check for it.

	pull := models.PullRequest{
		URL:        event.ObjectAttributes.URL,
		Author:     event.User.Username,
		Num:        event.ObjectAttributes.IID,
		HeadCommit: event.ObjectAttributes.LastCommit.ID,
		Branch:     event.ObjectAttributes.SourceBranch,
		State:      modelState,
	}

	cloneURL := e.addGitlabAuth(event.Project.GitHTTPURL)
	repo := models.Repo{
		FullName:          event.Project.PathWithNamespace,
		Name:              event.Project.Name,
		SanitizedCloneURL: event.Project.GitHTTPURL,
		Owner:             event.Project.Namespace,
		CloneURL:          cloneURL,
	}
	return pull, repo
}

// addGitlabAuth adds gitlab username/password to the cloneURL.
// Expects cloneURL to be an https URL.
// Ex. https://gitlab.com/owner/repo.git => https://uname:pass@gitlab.com/owner/repo.git
func (e *EventParser) addGitlabAuth(cloneURL string) string {
	return strings.Replace(cloneURL, "https://", fmt.Sprintf("https://%s:%s@", e.GitlabUser, e.GitlabToken), -1)
}

// ParseGitlabMergeCommentEvent creates Atlantis models out of a GitLab event.
func (e *EventParser) ParseGitlabMergeCommentEvent(event gitlab.MergeCommentEvent) (baseRepo models.Repo, headRepo models.Repo, user models.User) {
	baseRepo = models.Repo{
		FullName:          event.Project.PathWithNamespace,
		Name:              event.Project.Name,
		SanitizedCloneURL: event.Project.GitHTTPURL,
		Owner:             event.Project.Namespace,
		CloneURL:          e.addGitlabAuth(event.Project.GitHTTPURL),
	}
	user = models.User{
		Username: event.User.Username,
	}
	headRepo = models.Repo{
		FullName:          event.MergeRequest.Source.PathWithNamespace,
		Name:              event.MergeRequest.Source.Name,
		SanitizedCloneURL: event.MergeRequest.Source.GitHTTPURL,
		Owner:             event.MergeRequest.Source.Namespace,
		CloneURL:          e.addGitlabAuth(event.MergeRequest.Source.GitHTTPURL),
	}
	return
}

func (e *EventParser) ParseGitlabMergeRequest(mr *gitlab.MergeRequest) models.PullRequest {
	pullState := models.Closed
	if mr.State == "opened" {
		pullState = models.Open
	}
	// GitLab also has a "merged" state, but we map that to Closed so we don't
	// need to check for it.

	return models.PullRequest{
		URL:        mr.WebURL,
		Author:     mr.Author.Username,
		Num:        mr.IID,
		HeadCommit: mr.SHA,
		Branch:     mr.SourceBranch,
		State:      pullState,
	}
}

func (e *EventParser) stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// nolint: unparam
func (e *EventParser) removeOccurrences(a string, list []string) []string {
	var out []string
	for _, b := range list {
		if b != a {
			out = append(out, b)
		}
	}
	return out
}
