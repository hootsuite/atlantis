package server

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

const InlineShebang = "/bin/sh -e"

type PreRun struct {
	Commands         []string
	Path             string
	Environment      string
	TerraformVersion string
}

// Start is the function that starts the pre run
func (p *PreRun) Start() (string, error) {
	var execScript string

	// we create a script from the commands provided
	s, err := p.createScript(p.Commands)
	if err != nil {
		return "", err
	}
	execScript = s

	var output string
	if execScript != "" {
		defer os.Remove(execScript)
		// set environment variable for the run.
		// this is to support scripts to use the ENVIRONMENT, ATLANTIS_TERRAFORM_VERSION
		// and WORKSPACE variables in their scripts
		if p.Environment != "" {
			os.Setenv("ENVIRONMENT", p.Environment)
		}
		if p.TerraformVersion != "" {
			os.Setenv("ATLANTIS_TERRAFORM_VERSION", p.TerraformVersion)
		}
		os.Setenv("WORKSPACE", p.Path)

		return p.execute(execScript)
	}

	return output, nil

}

func (p *PreRun) createScript(cmds []string) (string, error) {
	var scriptName string
	if cmds != nil {
		tmp, err := ioutil.TempFile("/tmp", "atlantis-temp-script")
		if err != nil {
			return "", fmt.Errorf("Error preparing shell script: %s", err)
		}

		scriptName = tmp.Name()

		// Write our contents to it
		writer := bufio.NewWriter(tmp)
		writer.WriteString(fmt.Sprintf("#!%s\n", InlineShebang))
		for _, command := range cmds {
			if _, err := writer.WriteString(command + "\n"); err != nil {
				return "", fmt.Errorf("Error preparing script: %s", err)
			}
		}

		if err := writer.Flush(); err != nil {
			return "", fmt.Errorf("Error flushing file when preparing script: %s", err)
		}
		tmp.Close()
	}

	return scriptName, nil
}

func (p *PreRun) execute(script string) (string, error) {
	if _, err := os.Stat(script); err == nil {
		os.Chmod(script, 0775)
	}
	localCmd := exec.Command("sh", "-c", script)
	out, err := localCmd.CombinedOutput()
	output := string(out)
	if err != nil {
		return output, fmt.Errorf("Error running script %s: %v %s", script, err, output)
	}

	return output, nil
}
