package events

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hootsuite/atlantis/server/events/github"
	"github.com/hootsuite/atlantis/server/events/locking"
	"github.com/hootsuite/atlantis/server/events/models"
	"github.com/hootsuite/atlantis/server/events/run"
	"github.com/hootsuite/atlantis/server/events/terraform"
	"github.com/pkg/errors"
)

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_lock_url_generator.go LockURLGenerator

type LockURLGenerator interface {
	// SetLockURL takes a function that given a lock id, will return a url
	// to view that lock
	SetLockURL(func(id string) (url string))
}

// atlantisUserTFVar is the name of the variable we execute terraform
// with containing the github username of who is running the command
const atlantisUserTFVar = "atlantis_user"

// PlanExecutor handles everything related to running terraform plan
// including integration with S3, Terraform, and GitHub
type PlanExecutor struct {
	Github            github.Client
	Terraform         *terraform.Client
	Locker            locking.Locker
	LockURL           func(id string) (url string)
	Run               *run.Run
	Workspace         Workspace
	ProjectPreExecute *ProjectPreExecute
}

type PlanSuccess struct {
	TerraformOutput string
	LockURL         string
}

func (p *PlanExecutor) SetLockURL(f func(id string) (url string)) {
	p.LockURL = f
}

func (p *PlanExecutor) Execute(ctx *CommandContext) CommandResponse {
	// figure out what projects have been modified so we know where to run plan
	modifiedFiles, err := p.Github.GetModifiedFiles(ctx.BaseRepo, ctx.Pull)
	if err != nil {
		return CommandResponse{Error: errors.Wrap(err, "getting modified files")}
	}
	ctx.Log.Info("found %d files modified in this pull request", len(modifiedFiles))

	modifiedTerraformFiles := p.filterToTerraform(modifiedFiles)
	if len(modifiedTerraformFiles) == 0 {
		return CommandResponse{Failure: "No Terraform files were modified."}
	}
	ctx.Log.Info("filtered modified files to %d non-module .tf files: %v", len(modifiedTerraformFiles), modifiedTerraformFiles)

	projects := p.ModifiedProjects(ctx.BaseRepo.FullName, modifiedTerraformFiles)
	var paths []string
	for _, p := range projects {
		paths = append(paths, p.Path)
	}
	ctx.Log.Info("based on files modified, determined we have %d modified project(s) at path(s): %v", len(projects), strings.Join(paths, ", "))

	cloneDir, err := p.Workspace.Clone(ctx.Log, ctx.BaseRepo, ctx.HeadRepo, ctx.Pull, ctx.Command.Environment)
	if err != nil {
		return CommandResponse{Error: err}
	}

	results := []ProjectResult{}
	for _, project := range projects {
		ctx.Log.Info("running plan for project at path %q", project.Path)
		result := p.plan(ctx, cloneDir, project)
		result.Path = project.Path
		results = append(results, result)
	}
	return CommandResponse{ProjectResults: results}
}

// plan runs the steps necessary to run `terraform plan`. If there is an error, the error message will be encapsulated in error
// and the GeneratePlanResponse struct will also contain the full log including the error
func (p *PlanExecutor) plan(ctx *CommandContext, repoDir string, project models.Project) ProjectResult {
	preExecute := p.ProjectPreExecute.Execute(ctx, repoDir, project)
	if preExecute.ProjectResult != (ProjectResult{}) {
		return preExecute.ProjectResult
	}
	config := preExecute.ProjectConfig
	terraformVersion := preExecute.TerraformVersion
	tfEnv := ctx.Command.Environment

	// Run terraform plan
	planFile := filepath.Join(repoDir, project.Path, fmt.Sprintf("%s.tfplan", tfEnv))
	userVar := fmt.Sprintf("%s=%s", atlantisUserTFVar, ctx.User.Username)
	planExtraArgs := config.GetExtraArguments(ctx.Command.Name.String())
	tfPlanCmd := append(append([]string{"plan", "-refresh", "-no-color", "-out", planFile, "-var", userVar}, planExtraArgs...), ctx.Command.Flags...)

	// check if env/{environment}.tfvars exist
	tfEnvFileName := filepath.Join("env", tfEnv+".tfvars")
	if _, err := os.Stat(filepath.Join(repoDir, project.Path, tfEnvFileName)); err == nil {
		tfPlanCmd = append(tfPlanCmd, "-var-file", tfEnvFileName)
	}
	output, err := p.Terraform.RunCommandWithVersion(ctx.Log, filepath.Join(repoDir, project.Path), tfPlanCmd, terraformVersion, tfEnv)
	if err != nil {
		// plan failed so unlock the state
		if _, unlockErr := p.Locker.Unlock(preExecute.LockResponse.LockKey); unlockErr != nil {
			ctx.Log.Err("error unlocking state after plan error: %v", unlockErr)
		}
		return ProjectResult{Error: fmt.Errorf("%s\n%s", err.Error(), output)}
	}
	ctx.Log.Info("plan succeeded")

	// if there are post plan commands then run them
	if len(config.PostPlan) > 0 {
		absolutePath := filepath.Join(repoDir, project.Path)
		_, err := p.Run.Execute(ctx.Log, config.PostPlan, absolutePath, tfEnv, terraformVersion, "post_plan")
		if err != nil {
			return ProjectResult{Error: errors.Wrap(err, "running post plan commands")}
		}
	}

	return ProjectResult{
		PlanSuccess: &PlanSuccess{
			TerraformOutput: output,
			LockURL:         p.LockURL(preExecute.LockResponse.LockKey),
		},
	}
}

func (p *PlanExecutor) filterToTerraform(files []string) []string {
	var out []string
	for _, fileName := range files {
		if !p.isInExcludeList(fileName) && strings.Contains(fileName, ".tf") {
			out = append(out, fileName)
		}
	}
	return out
}

func (p *PlanExecutor) isInExcludeList(fileName string) bool {
	return strings.Contains(fileName, "terraform.tfstate") || strings.Contains(fileName, "terraform.tfstate.backup") || strings.Contains(fileName, "_modules") || strings.Contains(fileName, "modules")
}

// ModifiedProjects returns the list of Terraform projects that have been changed due to the
// modified files
func (p *PlanExecutor) ModifiedProjects(repoFullName string, modifiedFiles []string) []models.Project {
	var projects []models.Project
	seenPaths := make(map[string]bool)
	for _, modifiedFile := range modifiedFiles {
		path := p.getProjectPath(modifiedFile)
		if _, ok := seenPaths[path]; !ok {
			projects = append(projects, models.NewProject(repoFullName, path))
			seenPaths[path] = true
		}
	}
	return projects
}

// getProjectPath returns the path to the project relative to the repo root
// if the project is at the root returns "."
func (p *PlanExecutor) getProjectPath(modifiedFilePath string) string {
	dir := path.Dir(modifiedFilePath)
	if path.Base(dir) == "env" {
		// if the modified file was inside an env/ directory, we treat this specially and
		// run plan one level up
		return path.Dir(dir)
	}
	return dir
}
