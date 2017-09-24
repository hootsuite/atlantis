package server

import (
	"strings"

	version "github.com/hashicorp/go-version"
)

// populateRuntimeEnvironmentVariables populates the terraform extra vars specified in the project config file
// with atlantis specific environment variables
func populateRuntimeEnvironmentVariables(extraArgs []string, workspaceDir string, tfEnv string, tfVersion *version.Version) []string {
	var extraArgsFinal []string
	for _, v := range extraArgs {
		if strings.Contains(v, "${ENVIRONMENT}") || strings.Contains(v, "${ATLANTIS_TERRAFORM_VERSION}") || strings.Contains(v, "${WORKSPACE}") {
			v = strings.Replace(v, "${ENVIRONMENT}", tfEnv, -1)
			v = strings.Replace(v, "${ATLANTIS_TERRAFORM_VERSION}", tfVersion.String(), -1)
			v = strings.Replace(v, "${WORKSPACE}", workspaceDir, -1)
		}
		extraArgsFinal = append(extraArgsFinal, v)
	}
	return extraArgsFinal
}
