package typetags

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/sliceutil"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/steplib"
	stepmanModels "github.com/bitrise-io/stepman/models"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

var typeTags = []string{
	"access-control",
	"artifact-info",
	"installer",
	"deploy",
	"utility",
	"dependency",
	"code-sign",
	"build",
	"test",
	"notification",
}
var projectTypeTags = []string{
	"ios",
	"macos",
	"android",
	"xamarin",
	"react-native",
	"cordova",
	"ionic",
	"fastlane",
	"web",
}

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
			return fmt.Errorf("Failed to find step by ID: %s and version: %s", stepID, vers)
		}
		return validateStepYML(version.StepModel)
	}

	var errors []string
	for _, step := range sl.Steps {
		if err := validateStepYML(step.Latest.StepModel); err != nil {
			errors = append(errors, fmt.Sprintf(" - %s@%s:\n%s", step.ID, step.Latest.ID, err.Error()))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

// - %s@%s has invalid value_options
func validateStepYML(stepModel stepmanModels.StepModel) error {
	var errors []string

	for _, typeTag := range stepModel.TypeTags {
		if !sliceutil.IsStringInSlice(typeTag, typeTags) {
			errors = append(errors, fmt.Sprintf("   - invalid type_tag: %s", typeTag))
		}
	}

	for _, projectTypeTag := range stepModel.ProjectTypeTags {
		if !sliceutil.IsStringInSlice(projectTypeTag, projectTypeTags) {
			errors = append(errors, fmt.Sprintf("   - invalid project_type_tag: %s", projectTypeTag))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

// String will return a short summary of the validator task
func (v *Validator) String() string {
	return "Ensure all type_tags and project_type_tags are supported"
}
