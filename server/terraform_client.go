package server

import (
	"os/exec"
	"regexp"
	"github.com/pkg/errors"
	version "github.com/hashicorp/go-version"
	"fmt"
)

type TerraformClient struct {
	defaultVersion *version.Version
}

var terraformVersionRegex = regexp.MustCompile("Terraform v(.*)\n")

func NewTerraformClient() (*TerraformClient, error) {
	versionCmdOutput, err := exec.Command("terraform", "version").CombinedOutput()
	output := string(versionCmdOutput)
	if err != nil {
		if err == exec.ErrNotFound {
			return nil, errors.New("terraform not found in $PATH. Download terraform from https://www.terraform.io/downloads.html")
		}
		return nil, errors.Wrapf(err, "running terraform version: %s", output)
	}
	match := terraformVersionRegex.FindStringSubmatch(output)
	if len(match) <= 1 {
		return nil, fmt.Errorf("could not parse terraform version from %s", output)
	}
	version, err := version.NewVersion(match[1])
	if err != nil {
		return nil, errors.Wrap(err, "parsing terraform version")
	}

	return &TerraformClient{
		defaultVersion: version,
	}, nil
}

func (t *TerraformClient) RunTerraformCommand(path string, tfCmd []string, tfEnvVars []string) ([]string, string, error) {
	return t.RunTerraformCommandWithVersion(path, tfCmd, tfEnvVars, t.defaultVersion)
}

func (t *TerraformClient) Version() *version.Version {
	return t.defaultVersion
}

func (t *TerraformClient) RunTerraformCommandWithVersion(path string, tfCmd []string, tfEnvVars []string, v *version.Version) ([]string, string, error) {
	tfExecutable := "terraform"
	// if version is the same as the default, don't need to prepend the version name to the executable
	if !v.Equal(t.defaultVersion) {
		tfExecutable = fmt.Sprintf("%s%s", tfExecutable, v.String())
	}
	terraformCmd := exec.Command(tfExecutable, tfCmd...)
	terraformCmd.Dir = path
	if len(tfEnvVars) > 0 {
		terraformCmd.Env = tfEnvVars
	}
	out, err := terraformCmd.CombinedOutput()
	output := string(out)
	if err != nil {
		return terraformCmd.Args, output, err
	}

	return terraformCmd.Args, output, nil
}
