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

type Config struct {
	PrePlan          PrePlan  `yaml:"pre_plan"`
	PreApply         PreApply `yaml:"pre_apply"`
	TerraformVersion string   `yaml:"terraform_version"`
	TerraformArgs    []string `yaml:"terraform_args"`
}

func (c *Config) Exists(execPath string) bool {
	// Check if config file exists
	_, err := os.Stat(filepath.Join(execPath, ProjectConfigFile))
	return err == nil
}

func (c *Config) Read(execPath string) error {
	raw, err := ioutil.ReadFile(filepath.Join(execPath, ProjectConfigFile))
	if err != nil {
		return fmt.Errorf("Couldn't read config file %q: %v", execPath, err)
	}

	if err := yaml.Unmarshal(raw, &c); err != nil {
		return fmt.Errorf("Couldn't decode yaml in config file %q: %v", execPath, err)
	}

	return nil
}
