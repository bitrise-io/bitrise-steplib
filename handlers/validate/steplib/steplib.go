package steplib

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	version "github.com/hashicorp/go-version"
)

type StepLib struct {
	Steps []Step
}

func (sl StepLib) FindStepByID(id string) Step {
	for _, step := range sl.Steps {
		if step.ID == id {
			return step
		}
	}
	return Step{}
}

func NewStepLib(rootPath string) (StepLib, error) {
	stepLib := StepLib{}
	stepsDir := filepath.Join(rootPath, "steps")
	stepDirs, err := ioutil.ReadDir(stepsDir)
	if err != nil {
		return StepLib{}, nil
	}
	for _, stepDirInfo := range stepDirs {
		if !stepDirInfo.IsDir() {
			continue
		}
		stepDir := filepath.Join(stepsDir, stepDirInfo.Name())
		if exists, err := pathutil.IsPathExists(filepath.Join(stepDir, "step-info.yml")); err == nil && exists {
			continue
		}
		versionDirs, err := ioutil.ReadDir(stepDir)
		if err != nil {
			return StepLib{}, err
		}
		step := Step{
			ID: stepDirInfo.Name(),
		}
		for _, versionDirInfo := range versionDirs {
			if !versionDirInfo.IsDir() || versionDirInfo.Name() == "assets" {
				continue
			}
			version := Version{
				ID:          versionDirInfo.Name(),
				StepYMLPath: filepath.Join(stepDir, versionDirInfo.Name(), "step.yml"),
			}
			b, err := fileutil.ReadBytesFromFile(version.StepYMLPath)
			if err != nil {
				return StepLib{}, err
			}
			version.Raw = b

			if err := yaml.Unmarshal(b, &version.StepModel); err != nil {
				return StepLib{}, err
			}

			step.Versions = append(step.Versions, version)
		}

		latestVersion, err := findLatestVersion(step.Versions)
		if err != nil {
			return StepLib{}, err
		}
		step.Latest = latestVersion

		stepLib.Steps = append(stepLib.Steps, step)
	}

	return stepLib, nil
}

func findLatestVersion(versions []Version) (Version, error) {
	latestVersion, err := version.NewVersion("0.0.0")
	if err != nil {
		return Version{}, err
	}
	var versionFound Version
	for _, v := range versions {
		actualVersion, err := version.NewVersion(v.ID)
		if err != nil {
			return Version{}, err
		}
		if actualVersion.GreaterThan(latestVersion) {
			latestVersion = actualVersion
			versionFound = v
		}
	}
	if latestVersion.String() == "0.0.0" {
		return Version{}, fmt.Errorf("unable to find latest version")
	}
	return versionFound, nil
}
