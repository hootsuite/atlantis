package server

import (
	"testing"

	version "github.com/hashicorp/go-version"
	. "github.com/hootsuite/atlantis/testing_util"
)

func TestPopulateRuntimeEnvironmentVariables(t *testing.T) {
	// test environment variable ${ENVIRONMENT}
	extraArgs := []string{"-backend-config=env/${ENVIRONMENT}.tfvars", "-no-color"}
	expectedArgs := []string{"-backend-config=env/testing.tfvars", "-no-color"}
	tfVersion, _ := version.NewVersion("0.1.1")
	args := populateRuntimeEnvironmentVariables(extraArgs, "./workspace", "testing", tfVersion)
	Equals(t, expectedArgs, args)

	// test environment variable ${WORKSPACE}
	extraArgs = []string{"-from-module=${WORKSPACE}/module", "-no-color"}
	expectedArgs = []string{"-from-module=./path/to/workspace/module", "-no-color"}
	tfVersion, _ = version.NewVersion("0.1.1")
	args = populateRuntimeEnvironmentVariables(extraArgs, "./path/to/workspace", "testing", tfVersion)
	Equals(t, expectedArgs, args)

	// test environment variable ${ATLANTIS_TERRAFORM_VERSION}
	extraArgs = []string{"-backend-config=env/${ATLANTIS_TERRAFORM_VERSION}/testing.tfvars", "-no-color"}
	expectedArgs = []string{"-backend-config=env/0.1.1/testing.tfvars", "-no-color"}
	tfVersion, _ = version.NewVersion("0.1.1")
	args = populateRuntimeEnvironmentVariables(extraArgs, "./path/to/workspace", "testing", tfVersion)
	Equals(t, expectedArgs, args)

	// test all environment variables together
	extraArgs = []string{"-backend-config=${WORKSPACE}/env/${ATLANTIS_TERRAFORM_VERSION}/${ENVIRONMENT}.tfvars", "-no-color"}
	expectedArgs = []string{"-backend-config=./path/to/workspace/env/0.1.1/testing.tfvars", "-no-color"}
	tfVersion, _ = version.NewVersion("0.1.1")
	args = populateRuntimeEnvironmentVariables(extraArgs, "./path/to/workspace", "testing", tfVersion)
	Equals(t, expectedArgs, args)
}
