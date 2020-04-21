package valueoptions

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/steplib"
	"github.com/bitrise-tools/go-steputils/stepconf"

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
func (v *Validator) Validate(sl steplib.StepLib) error {
	if err := stepconf.Parse(&v.c); err != nil {
		return fmt.Errorf("Issue with input: %s", err)
	}

	if v.c.Step != "" {
		stepID := strings.Split(v.c.Step, "@")[0]
		vers := strings.Split(v.c.Step, "@")[1]

		version := sl.FindStepByID(stepID).FindVersionByID(vers)
		if version.ID == "" {
			return fmt.Errorf("Failed to find step by ID: %s and version: %s, steplib: %#v", stepID, vers, sl)
		}
		return validateStepYML(version.Raw)
	}

	var errors []string
	for _, step := range sl.Steps {
		if err := validateStepYML(step.Latest.Raw); err != nil {
			errors = append(errors, fmt.Sprintf(" - %s@%s:\n%s", step.ID, step.Latest.ID, err.Error()))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

func validateStepYML(data []byte) error {
	var errors []string

	var step struct {
		Inputs []struct {
			Opts struct {
				Title        string        `yaml:"title,omitempty"`
				ValueOptions []interface{} `yaml:"value_options,omitempty"`
			} `yaml:"opts,omitempty"`
		} `yaml:"inputs,omitempty"`
	}
	if err := yaml.Unmarshal(data, &step); err != nil {
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
				errors = append(errors, fmt.Sprintf("   - type(%s) in value_options with value: %v at input: %s", reflect.TypeOf(v), v, input.Opts.Title))
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
