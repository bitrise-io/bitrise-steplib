package collection

import (
	"encoding/base64"
	"fmt"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/steplib"
	"github.com/bitrise-io/go-utils/command"
)

const testBitriseYML = `format_version: "6"
default_step_lib_source: file://./

workflows:
  test:
    steps:
    - script:
        inputs:
        - content: echo "successful"
`

// Validator is matching for the validator interface
type Validator struct{}

// IsSkippable sets the validation task to skip failures
func (v *Validator) IsSkippable() bool {
	return false
}

// Validate is the logic handler of the task
func (v *Validator) Validate(sl steplib.StepLib) error {
	//base64.StdEncoding.EncodeToString(
	out, err := command.New("bitrise", "run", "--config-base64", base64.StdEncoding.EncodeToString([]byte(testBitriseYML)), "test").RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf(" - CLI run failed, output:\n%s\n\nerror: %s", out, err)
	}
	return nil
}

// String will return a short summary of the validator task
func (v *Validator) String() string {
	return "Ensure CLI is working properly with the new collection"
}
