package server

import (
	"path/filepath"
	"strconv"
	"os"
	"github.com/pkg/errors"
	"os/exec"
	"fmt"
	"io/ioutil"
	"github.com/hootsuite/atlantis/models"
)

const defaultSSHWrapper = "/tmp/git-ssh.sh"

type Workspace struct{
	scratchDir string
	sshKey string
}

func (w *Workspace) Clone(ctx *CommandContext) (string, error) {
	cloneDir := w.cloneDir(ctx)

	// this is safe to do because we lock runs on repo/pull/env so no one else is using this workspace
	ctx.Log.Info("cleaning clone directory %q", cloneDir)
	if err := os.RemoveAll(cloneDir); err != nil {
		return "", errors.Wrap(err, "deleting old workspace")
	}

	// create the directory and parents if necessary
	ctx.Log.Info("creating dir %q", cloneDir)
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		return "", errors.Wrap(err, "creating new workspace")
	}

	// Check if ssh key is set and create git ssh wrapper
	cloneCmd := exec.Command("git", "clone", ctx.Repo.SSHURL, cloneDir)
	if w.sshKey != "" {
		// todo: is this still needed?
		err := w.generateSSHWrapper()
		if err != nil {
			return "", errors.Wrap(err, "creating git ssh wrapper")
		}
		cloneCmd.Env = []string{
			fmt.Sprintf("GIT_SSH=%s", defaultSSHWrapper),
			fmt.Sprintf("PKEY=%s", w.sshKey),
		}
	}

	// clone the repo
	ctx.Log.Info("git cloning %q into %q", ctx.Repo.SSHURL, cloneDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return "", errors.Wrapf(err,"cloning %s: %s", ctx.Repo.SSHURL, string(output))
	}

	// check out the branch for this PR
	ctx.Log.Info("checking out branch %q", ctx.Pull.Branch)
	checkoutCmd := exec.Command("git", "checkout", ctx.Pull.Branch)
	checkoutCmd.Dir = cloneDir
	if err := checkoutCmd.Run(); err != nil {
		return "", errors.Wrapf(err,"checking out branch %s", ctx.Pull.Branch)
	}
	return cloneDir, nil
}

func (w *Workspace) GetWorkspace(ctx *CommandContext) (string, error) {
	repoDir := w.cloneDir(ctx)
	if _, err := os.Stat(repoDir); err != nil {
		return "", errors.Wrap(err, "checking if workspace exists")
	}
	return repoDir, nil
}

// CleanProject deletes all .terraform/ folders and *.tfplan files from dir
func (w *Workspace) CleanProject(dir string) error {
	// delete .terraform directories
	if err := os.RemoveAll(filepath.Join(dir, ".terraform")); err != nil {
		return errors.Wrap(err, "cleaning old .terraform directories")
	}

	// delete old plan files
	ms, err := filepath.Glob(fmt.Sprintf("%s/*.tfplan", filepath.Clean(dir)))
	if err != nil {
		return errors.Wrap(err, "finding .tfplan files")
	}
	for _, m := range ms {
		if err := os.Remove(m); err != nil {
			return errors.Wrap(err, "delete .tfplan file")
		}
	}
	return nil
}

// Delete deletes the workspace for this repo and pull
func (w *Workspace) Delete(repo models.Repo, pull models.PullRequest) error {
	return os.RemoveAll(w.repoPullDir(repo, pull))
}

func (w *Workspace) repoPullDir(repo models.Repo, pull models.PullRequest) string {
	return filepath.Join(w.scratchDir, repo.FullName, strconv.Itoa(pull.Num))
}

func (w *Workspace) cloneDir(ctx *CommandContext) string {
	return filepath.Join(w.repoPullDir(ctx.Repo, ctx.Pull), ctx.Command.environment)
}

func (w *Workspace) generateSSHWrapper() error {
	d1 := []byte("#!/bin/sh\nif [ -z \"$PKEY\" ]; then\n# if PKEY is not specified, run ssh using default keyfile\nssh -oStrictHostKeyChecking=no \"$@\"\nelse\nssh -oStrictHostKeyChecking=no -i \"$PKEY\" \"$@\"\nfi")
	return ioutil.WriteFile(defaultSSHWrapper, d1, 0755)
}
