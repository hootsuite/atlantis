// Package terraform handles the actual running of terraform commands
package terraform

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hootsuite/atlantis/server/logging"
	"github.com/pkg/errors"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_runner.go Runner
type Runner interface {
	Version() *version.Version
	RunCommandWithVersion(log *logging.SimpleLogger, path string, args []string, v *version.Version, env string) (string, error)
	RunInitAndWorkspace(log *logging.SimpleLogger, path string, env string, extraInitArgs []string, version *version.Version) ([]string, error)
}

type Client struct {
	defaultVersion *version.Version
}

var versionRegex = regexp.MustCompile("Terraform v(.*)\n")

func NewClient() (*Client, error) {
	// may be use exec.LookPath?
	versionCmdOutput, err := exec.Command("terraform", "version").CombinedOutput()
	output := string(versionCmdOutput)
	if err != nil {
		// exec.go line 35, Error() returns
		// "exec: " + strconv.Quote(e.Name) + ": " + e.Err.Error()
		if err.Error() == fmt.Sprintf("exec: \"terraform\": %s", exec.ErrNotFound.Error()) {
			return nil, errors.New("terraform not found in $PATH. \n\nDownload terraform from https://www.terraform.io/downloads.html")
		}
		return nil, errors.Wrapf(err, "running terraform version: %s", output)
	}
	match := versionRegex.FindStringSubmatch(output)
	if len(match) <= 1 {
		return nil, fmt.Errorf("could not parse terraform version from %s", output)
	}
	version, err := version.NewVersion(match[1])
	if err != nil {
		return nil, errors.Wrap(err, "parsing terraform version")
	}

	return &Client{
		defaultVersion: version,
	}, nil
}

// Version returns the version of the terraform executable in our $PATH.
func (c *Client) Version() *version.Version {
	return c.defaultVersion
}

// RunCommandWithVersion executes the provided version of terraform with
// the provided args in path. The variable "v" is the version of terraform executable to use and the variable "env" is the
// environment specified by the user commenting "atlantis plan/apply {env}" which is set to "default" by default.
func (c *Client) RunCommandWithVersion(log *logging.SimpleLogger, path string, args []string, v *version.Version, env string) (string, error) {
	tfExecutable := "terraform"
	// if version is the same as the default, don't need to prepend the version name to the executable
	if !v.Equal(c.defaultVersion) {
		tfExecutable = fmt.Sprintf("%s%s", tfExecutable, v.String())
	}

	// set environment variables
	// this is to support scripts to use the ENVIRONMENT, ATLANTIS_TERRAFORM_VERSION
	// and WORKSPACE variables in their scripts
	// append current process's environment variables
	// this is to prevent the $PATH variable being removed from the environment
	envVars := []string{
		fmt.Sprintf("ENVIRONMENT=%s", env),
		fmt.Sprintf("ATLANTIS_TERRAFORM_VERSION=%s", v.String()),
		fmt.Sprintf("WORKSPACE=%s", path),
	}
	envVars = append(envVars, os.Environ()...)

	// append terraform executable name with args
	tfCmd := fmt.Sprintf("%s %s", tfExecutable, strings.Join(args, " "))

	terraformCmd := exec.Command("sh", "-c", tfCmd)
	terraformCmd.Dir = path
	terraformCmd.Env = envVars
	out, err := terraformCmd.CombinedOutput()
	commandStr := strings.Join(terraformCmd.Args, " ")
	if err != nil {
		err := fmt.Errorf("%s: running %q in %q: \n%s", err, commandStr, path, out)
		log.Debug("error: %s", err)
		return string(out), err
	}
	log.Info("successfully ran %q in %q", commandStr, path)
	return string(out), nil
}

// RunInitAndWorkspace executes the following:
// 1. "terraform init" - This initializes terraform project
// 2. "terraform workspace or env select" - This selects the workspace or environment for the terraform project
// [optional] 3. "terraform workspace or env new" - This creates a new workspace or environment for the terraform project
// env is the environment supplied by the atlantis user that is used to
// select or create a new workspace or environment for terraform
func (c *Client) RunInitAndWorkspace(log *logging.SimpleLogger, path string, env string, extraInitArgs []string, v *version.Version) ([]string, error) {
	var outputs []string
	// run terraform init
	output, err := c.RunCommandWithVersion(log, path, append([]string{"init", "-no-color"}, extraInitArgs...), v, env)
	outputs = append(outputs, output)
	if err != nil {
		return outputs, err
	}

	// Terraform uses 'terraform env' command for versions > 0.8 and < 0.10.
	// Versions >= 0.10 use 'terraform workspace'
	workspaceCmdName := "workspace"
	constraints, _ := version.NewConstraint("< 0.10.0")
	if constraints.Check(v) {
		workspaceCmdName = "env"
	}

	// Run 'terraform workspace/env select'
	output, err = c.RunCommandWithVersion(log, path, []string{workspaceCmdName, "select", "-no-color", env}, v, env)
	outputs = append(outputs, output)
	if err != nil {
		// If 'terraform workspace/env select' fails we will run 'terraform workspace new'
		// to create a new environment. This is done for ease of use so that the atlantis
		// user doesn't have to create a new workspace/env manually.
		output, err = c.RunCommandWithVersion(log, path, []string{workspaceCmdName, "new", "-no-color", env}, v, env)
		outputs = append(outputs, output)
		if err != nil {
			return outputs, err
		}
	}
	return outputs, nil
}
