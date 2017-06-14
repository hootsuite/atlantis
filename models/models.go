package models

import "time"

type TFProject struct {
	RepoFullName string // could just have Repo object
	Path         string
}

type Run struct {
	TFProject   TFProject
	Environment string
	PullNum     int
}

type Repo struct {
	FullName string
	SSHURL string
}

type PullRequest struct {
	Num        int
	HeadCommit string
	BaseCommit string
	Link       string
	Branch     string
	Author     string
}

type User struct {
	Username string
	Email    string
}

type Lock struct {
	Run Run
	User User
	Time time.Time
}
