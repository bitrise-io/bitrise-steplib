package valueoptions

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/bitrise-tools/go-steputils/stepconf"
	version "github.com/hashicorp/go-version"
	yaml "gopkg.in/yaml.v2"
)

// Config ...
type Config struct {
	Step string `env:"step"`
}

// Validator is matching for the validator interface
type Validator struct {
	c Config
}

// IsSkippable sets the validation task to skip failures
func (v *Validator) IsSkippable() bool {
	return v.c.Step == ""
}

// Validate is the logic handler of the task
func (v *Validator) Validate() error {
	if err := stepconf.Parse(&v.c); err != nil {
		return fmt.Errorf("Issue with input: %s", err)
	}

	if v.c.Step != "" {
		stepID := strings.Split(v.c.Step, "@")[0]
		vers := strings.Split(v.c.Step, "@")[1]
		return validateStepYML(vers, stepID)
	}

	var errors []string
	stepLatestVersions := map[string]*version.Version{}
	if err := filepath.Walk("steps", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || info.Name() != "step.yml" {
			return nil
		}

		files, err := ioutil.ReadDir(filepath.Dir(filepath.Dir(path)))
		if err != nil {
			return err
		}

		// find out step id and version from path
		vers := filepath.Base(filepath.Dir(path))
		stepID := filepath.Base(filepath.Dir(filepath.Dir(path)))

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

		if vers == latestVersion.String() {
			if err := validateStepYML(vers, stepID); err != nil {
				errors = append(errors, err.Error())
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

func validateStepYML(vers, stepID string) error {
	path := filepath.Join("steps", stepID, vers, "step.yml")
	var errors []string

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
				errors = append(errors, fmt.Sprintf("  - %s@%s has invalid type(%s) in value_options with value: %v at input: %s", stepID, vers, reflect.TypeOf(v), v, input.Opts.Title))
				break
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

// String will return a short summary of the validator task
func (v *Validator) String() string {
	return "Ensure all value_option is string type"
}
