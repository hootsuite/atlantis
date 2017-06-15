package models

import "time"

type Project struct {
	Repo Repo
	Path string
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

type ProjectLock struct {
	Project Project
	Pull PullRequest
	Time time.Time
}
