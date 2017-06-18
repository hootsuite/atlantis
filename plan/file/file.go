package file

import (
	"path/filepath"
	"os"
	"github.com/hootsuite/atlantis/models"
	"github.com/pkg/errors"
	"strconv"
	"github.com/hootsuite/atlantis/plan"
	"io/ioutil"
)

type Backend struct {
	// baseDir is the root at which all plans will be stored
	baseDir string
}

func New(baseDir string) (*Backend, error) {
	baseDir = filepath.Clean(baseDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Backend{baseDir}, nil
}


// save plans to baseDir/owner/repo/pullNum/path/env.tfplan
func (b *Backend) SavePlan(path string, project models.Project, env string, pullNum int) error {
	savePath := b.path(project, pullNum)
	if err := os.MkdirAll(savePath, 0755); err != nil {
		return errors.Wrap(err, "creating save directory")
	}
	if err := b.copy(path, filepath.Join(savePath, env + ".tfplan")); err != nil {
		return errors.Wrap(err, "saving plan")
	}
	return nil
}

func (b *Backend) CopyPlans(dstRepo string, repoFullName string, env string, pullNum int) ([]plan.Plan, error) {
	// Look in the directory for this repo/pull and get plans for all projects.
	// Then filter to the plans for this environment
	var toCopy []string // will contain paths to the plan files relative to repo root
	root := filepath.Join(b.baseDir, repoFullName, strconv.Itoa(pullNum))
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// if the plan is for the right env,
		if info.Name() == env + ".tfplan" {
			rel, err := filepath.Rel(root, path)
			if err == nil {
				toCopy = append(toCopy, rel)
			}
		}
		return nil
	})

	var plans []plan.Plan
	if err != nil {
		return plans, errors.Wrap(err, "listing plans")
	}

	// copy the plans to the destination repo
	for _, file := range toCopy {
		dst := filepath.Join(dstRepo, file)
		if err := b.copy(filepath.Join(root, file), dst); err != nil {
			return plans, errors.Wrap(err, "copying plan")
		}
		plans = append(plans, plan.Plan{
			Project: models.Project{
				Path: filepath.Dir(file),
				RepoFullName: repoFullName,
			},
			LocalPath: dst,
		})
	}
	return plans, nil
}

func (b *Backend) copy(src string, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return errors.Wrapf(err, "reading %s", src)
	}

	if err = ioutil.WriteFile(dst, data, 0644); err != nil {
		return errors.Wrapf(err, "writing %s", dst)
	}
	return nil
}

func (b *Backend) path(p models.Project, pullNum int) string {
	return filepath.Join(b.baseDir, p.RepoFullName, strconv.Itoa(pullNum), p.Path)
}
