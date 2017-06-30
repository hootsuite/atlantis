package prerun

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

const InlineShebang = "/bin/sh -e"

type PreRun struct {
	Commands         []string
	Path             string
	Environment      string
	TerraformVersion string
}

func New(commands []string, path string, environment string, terraformVersion string) *PreRun {
	return &PreRun{
		Commands:         commands,
		Path:             path,
		Environment:      environment,
		TerraformVersion: terraformVersion,
	}
}

// todo: pass in all the args from New into Start and make this object stateless
// Start is the function that starts the pre run
func (p *PreRun) Start() (string, error) {
	// we create a script from the commands provided
	s, err := createScript(p.Commands)

	if err != nil {
		return "", err
	}

	defer os.Remove(s)
	// set environment variable for the run.
	// this is to support scripts to use the ENVIRONMENT, ATLANTIS_TERRAFORM_VERSION
	// and WORKSPACE variables in their scripts
	os.Setenv("ENVIRONMENT", p.Environment)
	os.Setenv("ATLANTIS_TERRAFORM_VERSION", p.TerraformVersion)
	os.Setenv("WORKSPACE", p.Path)
	return execute(s)
}

func createScript(cmds []string) (string, error) {
	// todo: use errors.Wrap
	// todo: ensure script is 755 here
	var scriptName string
	if cmds != nil {
		tmp, err := ioutil.TempFile("/tmp", "atlantis-temp-script")
		if err != nil {
			return "", fmt.Errorf("Error preparing shell script: %s", err)
		}

		scriptName = tmp.Name()

		// Write our contents to it
		// todo: confirm we need to do writestring and flush, is there a way to do this all at once?
		writer := bufio.NewWriter(tmp)
		writer.WriteString(fmt.Sprintf("#!%s\n", InlineShebang))
		cmdsJoined := strings.Join(cmds, "\n")
		if _, err := writer.WriteString(cmdsJoined); err != nil {
			return "", errors.Wrap(err, "preparing pre run")
		}

		if err := writer.Flush(); err != nil {
			return "", fmt.Errorf("Error flushing file when preparing script: %s", err)
		}
		tmp.Close()
	}

	return scriptName, nil
}

func execute(script string) (string, error) {
	localCmd := exec.Command("sh", "-c", script)
	out, err := localCmd.CombinedOutput()
	output := string(out)
	// todo: errors.Wrap
	if err != nil {
		return output, fmt.Errorf("Error running script %s: %v %s", script, err, output)
	}

	return output, nil
}
