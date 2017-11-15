package events_test

import (
	"testing"

	"errors"
	"strings"

	"github.com/google/go-github/github"
	"github.com/hootsuite/atlantis/server/events"
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/events/vcs"
	. "github.com/hootsuite/atlantis/server/events/vcs/fixtures"
	. "github.com/hootsuite/atlantis/testing"
	"github.com/mohae/deepcopy"
)

var parser = events.EventParser{
	GithubUser:  "github-user",
	GithubToken: "github-token",
	GitlabUser:  "gitlab-user",
	GitlabToken: "gitlab-token",
}

func TestDetermineCommandInvalid(t *testing.T) {
	t.Log("given a comment that does not match the regex should return an error")
	comments := []string{
		// just the executable, no command
		"run",
		"atlantis",
		"@github-user",
		// invalid command
		"run slkjd",
		"atlantis slkjd",
		"@github-user slkjd",
		"atlantis plans",
		// misc
		"related comment mentioning atlantis",
	}
	for _, c := range comments {
		_, e := parser.DetermineCommand(c, vcs.Github)
		Assert(t, e != nil, "expected error for comment: "+c)
	}
}

func TestDetermineCommandHelp(t *testing.T) {
	t.Log("given a help comment, should match")
	comments := []string{
		"run help",
		"atlantis help",
		"@github-user help",
		"atlantis help --verbose",
	}
	for _, c := range comments {
		command, e := parser.DetermineCommand(c, vcs.Github)
		Ok(t, e)
		Equals(t, events.Help, command.Name)
	}
}

func TestDetermineCommandPermutations(t *testing.T) {
	execNames := []string{"run", "atlantis", "@github-user", "@gitlab-user"}
	commandNames := []events.CommandName{events.Plan, events.Apply}
	envs := []string{"", "default", "env", "env-dash", "env_underscore", "camelEnv"}
	flagCases := [][]string{
		{},
		{"--verbose"},
		{"-key=value"},
		{"-key", "value"},
		{"-key1=value1", "-key2=value2"},
		{"-key1=value1", "-key2", "value2"},
		{"-key1", "value1", "-key2=value2"},
		{"--verbose", "key2=value2"},
		{"-key1=value1", "--verbose"},
	}

	// test all permutations
	for _, exec := range execNames {
		for _, name := range commandNames {
			for _, env := range envs {
				for _, flags := range flagCases {
					// If github comments end in a newline they get \r\n appended.
					// Ensure that we parse commands properly either way.
					for _, lineEnding := range []string{"", "\r\n"} {
						comment := strings.Join(append([]string{exec, name.String(), env}, flags...), " ") + lineEnding
						t.Log("testing comment: " + comment)

						// In order to test gitlab without fully refactoring this test
						// we're just detecting if we're using the gitlab user as the
						// exec name.
						vcsHost := vcs.Github
						if exec == "@gitlab-user" {
							vcsHost = vcs.Gitlab
						}
						c, err := parser.DetermineCommand(comment, vcsHost)
						Ok(t, err)
						Equals(t, name, c.Name)
						if env == "" {
							Equals(t, "default", c.Environment)
						} else {
							Equals(t, env, c.Environment)
						}
						Equals(t, containsVerbose(flags), c.Verbose)

						// ensure --verbose never shows up in flags
						for _, f := range c.Flags {
							Assert(t, f != "--verbose", "Should not pass on the --verbose flag: %v", flags)
						}

						// check all flags are present
						for _, f := range flags {
							if f != "--verbose" {
								Contains(t, f, c.Flags)
							}
						}
					}
				}
			}
		}
	}
}

func TestParseGithubRepo(t *testing.T) {
	testRepo := Repo
	testRepo.FullName = nil
	_, err := parser.ParseGithubRepo(&testRepo)
	Equals(t, errors.New("repository.full_name is null"), err)

	testRepo = Repo
	testRepo.Owner = nil
	_, err = parser.ParseGithubRepo(&testRepo)
	Equals(t, errors.New("repository.owner.login is null"), err)

	testRepo = Repo
	testRepo.Name = nil
	_, err = parser.ParseGithubRepo(&testRepo)
	Equals(t, errors.New("repository.name is null"), err)

	testRepo = Repo
	testRepo.CloneURL = nil
	_, err = parser.ParseGithubRepo(&testRepo)
	Equals(t, errors.New("repository.clone_url is null"), err)

	t.Log("should replace https clone with user/pass")
	{
		r, err := parser.ParseGithubRepo(&Repo)
		Ok(t, err)
		Equals(t, models.Repo{
			Owner:             "owner",
			FullName:          "owner/repo",
			CloneURL:          "https://github-user:token@github.com/lkysow/atlantis-example.git",
			SanitizedCloneURL: Repo.GetCloneURL(),
			Name:              "repo",
		}, r)
	}
}

func TestParseGithubIssueCommentEvent(t *testing.T) {
	comment := github.IssueCommentEvent{
		Repo: &Repo,
		Issue: &github.Issue{
			Number:  github.Int(1),
			User:    &github.User{Login: github.String("issue_user")},
			HTMLURL: github.String("https://github.com/hootsuite/atlantis/issues/1"),
		},
		Comment: &github.IssueComment{
			User: &github.User{Login: github.String("comment_user")},
		},
	}
	testComment := deepcopy.Copy(comment).(github.IssueCommentEvent)
	testComment.Repo = nil
	_, _, _, err := parser.ParseGithubIssueCommentEvent(&testComment)
	Equals(t, errors.New("repository.full_name is null"), err)

	testComment = deepcopy.Copy(comment).(github.IssueCommentEvent)
	testComment.Issue = nil
	_, _, _, err = parser.ParseGithubIssueCommentEvent(&testComment)
	Equals(t, errors.New("issue.number is null"), err)

	testComment = deepcopy.Copy(comment).(github.IssueCommentEvent)
	testComment.Issue.User = nil
	_, _, _, err = parser.ParseGithubIssueCommentEvent(&testComment)
	Equals(t, errors.New("issue.user.login is null"), err)

	testComment = deepcopy.Copy(comment).(github.IssueCommentEvent)
	testComment.Issue.HTMLURL = nil
	_, _, _, err = parser.ParseGithubIssueCommentEvent(&testComment)
	Equals(t, errors.New("issue.html_url is null"), err)

	testComment = deepcopy.Copy(comment).(github.IssueCommentEvent)
	testComment.Comment.User.Login = nil
	_, _, _, err = parser.ParseGithubIssueCommentEvent(&testComment)
	Equals(t, errors.New("comment.user.login is null"), err)

	// this should be successful
	repo, user, pull, err := parser.ParseGithubIssueCommentEvent(&comment)
	Ok(t, err)
	Equals(t, models.Repo{
		Owner:             *comment.Repo.Owner.Login,
		FullName:          *comment.Repo.FullName,
		CloneURL:          "https://user:token@github.com/lkysow/atlantis-example.git",
		SanitizedCloneURL: *comment.Repo.CloneURL,
		Name:              "repo",
	}, repo)
	Equals(t, models.User{
		Username: *comment.Comment.User.Login,
	}, user)
	Equals(t, models.PullRequest{
		Num: *comment.Issue.Number,
	}, pull)
}

func TestParseGithubPull(t *testing.T) {
	testPull := deepcopy.Copy(Pull).(github.PullRequest)
	testPull.Head.SHA = nil
	_, _, err := parser.ParseGithubPull(&testPull)
	Equals(t, errors.New("head.sha is null"), err)

	testPull = deepcopy.Copy(Pull).(github.PullRequest)
	testPull.Base.SHA = nil
	_, _, err = parser.ParseGithubPull(&testPull)
	Equals(t, errors.New("base.sha is null"), err)

	testPull = deepcopy.Copy(Pull).(github.PullRequest)
	testPull.HTMLURL = nil
	_, _, err = parser.ParseGithubPull(&testPull)
	Equals(t, errors.New("html_url is null"), err)

	testPull = deepcopy.Copy(Pull).(github.PullRequest)
	testPull.Head.Ref = nil
	_, _, err = parser.ParseGithubPull(&testPull)
	Equals(t, errors.New("head.ref is null"), err)

	testPull = deepcopy.Copy(Pull).(github.PullRequest)
	testPull.User.Login = nil
	_, _, err = parser.ParseGithubPull(&testPull)
	Equals(t, errors.New("user.login is null"), err)

	testPull = deepcopy.Copy(Pull).(github.PullRequest)
	testPull.Number = nil
	_, _, err = parser.ParseGithubPull(&testPull)
	Equals(t, errors.New("number is null"), err)

	testPull = deepcopy.Copy(Pull).(github.PullRequest)
	testPull.Head.Repo = nil
	_, _, err = parser.ParseGithubPull(&testPull)
	Equals(t, errors.New("repository.full_name is null"), err)

	PullRes, repoRes, err := parser.ParseGithubPull(&Pull)
	Ok(t, err)
	Equals(t, models.PullRequest{
		URL:        Pull.GetHTMLURL(),
		Author:     Pull.User.GetLogin(),
		Branch:     Pull.Head.GetRef(),
		HeadCommit: Pull.Head.GetSHA(),
		Num:        Pull.GetNumber(),
	}, PullRes)

	Equals(t, models.Repo{
		Owner:             "owner",
		FullName:          "owner/repo",
		CloneURL:          "https://user:token@github.com/lkysow/atlantis-example.git",
		SanitizedCloneURL: Repo.GetCloneURL(),
		Name:              "repo",
	}, repoRes)
}

func containsVerbose(list []string) bool {
	for _, b := range list {
		if b == "--verbose" {
			return true
		}
	}
	return false
}
