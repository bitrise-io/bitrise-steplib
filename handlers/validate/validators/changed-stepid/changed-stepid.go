package typetags

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/steplib"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-tools/go-steputils/stepconf"
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
			return fmt.Errorf("Failed to find step by ID: %s and version: %s", stepID, vers)
		}
		return checkDuplicatedStepIDs(version, sl.Steps)
	}

	var errors []string
	for _, step := range sl.Steps {
		if err := checkDuplicatedStepIDs(step.Latest, sl.Steps); err != nil {
			errors = append(errors, fmt.Sprintf(" - %s", err.Error()))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

func checkDuplicatedStepIDs(step steplib.Version, steps []steplib.Step) error {
	var errors []string

	var alreadyChecked []string
	for _, s := range steps {
		if sliceutil.IsStringInSlice(step.StepID, alreadyChecked) {
			continue
		}
		if s.Latest.StepModel.Source.Git == step.StepModel.Source.Git && s.Latest.StepID != step.StepID {
			errors = append(errors, fmt.Sprintf("[%s] and [%s] steps are using the same git source: [%s]", s.Latest.StepID, step.StepID, step.StepModel.Source.Git))
		}
		alreadyChecked = append(alreadyChecked, s.Latest.StepID)
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

// String will return a short summary of the validator task
func (v *Validator) String() string {
	return "Check if same git repo has more then one step ID in StepLib"
}
