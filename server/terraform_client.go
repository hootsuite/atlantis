package server

import "os/exec"

type TerraformClient struct {
	tfExecutableName string
	tfVersion        string
}

func (t *TerraformClient) RunTerraformCommand(path string, tfCmd []string, tfEnvVars []string) ([]string, string, error) {
	terraformCmd := exec.Command(t.tfExecutableName, tfCmd...)
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
