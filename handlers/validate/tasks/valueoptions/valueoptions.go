package valueoptions

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	version "github.com/hashicorp/go-version"
	yaml "gopkg.in/yaml.v2"
)

// Task is matching for the validator interface
type Task struct{}

// IsSkippable sets the validation task to skip failures
func (t Task) IsSkippable() bool {
	return true
}

// Work is the logic handler of the task
func (t Task) Work() error {
	var errors []string
	stepLatestVersions := map[string]*version.Version{}
	err := filepath.Walk("steps", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || info.Name() != "step.yml" {
			return nil
		}

		// find out step id and version from path
		vers := filepath.Base(filepath.Dir(path))
		stepID := filepath.Base(filepath.Dir(filepath.Dir(path)))

		files, err := ioutil.ReadDir(filepath.Dir(filepath.Dir(path)))
		if err != nil {
			return err
		}

		latestVersion, err := version.NewVersion("0.0.0")
		if err != nil {
			return err
		}

		if lv, ok := stepLatestVersions[stepID]; !ok {
			for _, file := range files {
				cVersion, err := version.NewVersion(file.Name())
				if err != nil {
					continue
				}

				if latestVersion.LessThan(cVersion) {
					latestVersion = cVersion
				}
			}
		} else {
			latestVersion = lv
		}

		stepLatestVersions[stepID] = latestVersion

		// read step.yml
		stepYMLFile, err := os.Open(path)
		if err != nil {
			return err
		}

		var step struct {
			Inputs []struct {
				Opts struct {
					Title        string        `yaml:"title,omitempty"`
					ValueOptions []interface{} `yaml:"value_options,omitempty"`
				} `yaml:"opts,omitempty"`
			} `yaml:"inputs,omitempty"`
		}
		if err := yaml.NewDecoder(stepYMLFile).Decode(&step); err != nil {
			return err
		}

		for _, input := range step.Inputs {
			for _, v := range input.Opts.ValueOptions {
				switch v.(type) {
				case string:
					// all ok
					break
				default:
					// error only if the latest version affected
					if vers == latestVersion.String() {
						errors = append(errors, fmt.Sprintf("  - %s@%s has invalid type(%s) in value_options at input: %s", stepID, vers, reflect.TypeOf(v), input.Opts.Title))
					}
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

// String will return a short summary of the validator task
func (t Task) String() string {
	return "Ensure all value_option is string type"
}
