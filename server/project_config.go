package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

const ProjectConfigFile = "atlantis.yaml"

type PrePlan struct {
	Commands []string `yaml:"commands"`
}

type PreApply struct {
	Commands []string `yaml:"commands"`
}

type ProjectConfig struct {
	PrePlan          PrePlan                 `yaml:"pre_plan"`
	PreApply         PreApply                `yaml:"pre_apply"`
	TerraformVersion string                  `yaml:"terraform_version"`
	ExtraArguments   []CommandExtraArguments `yaml:"extra_arguments"`
}

type CommandExtraArguments struct {
	Name      string   `yaml:"command_name"`
	Arguments []string `yaml:"arguments"`
}

func (c *ProjectConfig) Exists(execPath string) bool {
	// Check if config file exists
	_, err := os.Stat(filepath.Join(execPath, ProjectConfigFile))
	return err == nil
}

func (c *ProjectConfig) Read(execPath string) error {
	raw, err := ioutil.ReadFile(filepath.Join(execPath, ProjectConfigFile))
	if err != nil {
		return fmt.Errorf("Couldn't read config file %q: %v", execPath, err)
	}

	if err := yaml.Unmarshal(raw, &c); err != nil {
		return fmt.Errorf("Couldn't decode yaml in config file %q: %v", execPath, err)
	}

	return nil
}

func (c *ProjectConfig) GetExtraArguments(command string) []string {
	for _, value := range c.ExtraArguments {
		if value.Name == command {
			return value.Arguments
		}
	}
	return []string{}
}
