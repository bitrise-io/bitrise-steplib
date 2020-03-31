package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bitrise-io/stepman/models"
	"github.com/qri-io/jsonschema"
)

// Model ...
type Model struct {
	spec []byte
}

// New ...
func New(specReader io.Reader) (Model, error) {
	spec, err := ioutil.ReadAll(specReader)
	if err != nil {
		return Model{}, err
	}

	return Model{spec: spec}, nil
}

// ValidateSchema ...
func (m Model) ValidateSchema(schemaReader io.Reader) ([]error, error) {
	schema, err := ioutil.ReadAll(schemaReader)
	if err != nil {
		return nil, err
	}

	rs := &jsonschema.RootSchema{}
	if err := json.Unmarshal(schema, rs); err != nil {
		return nil, err
	}

	valErrs, err := rs.ValidateBytes(m.spec)
	if err != nil {
		return nil, err
	}

	return toErrors(valErrs), nil
}

// ValidateTags ...
func (m Model) ValidateTags() ([]error, error) {
	var stepLib models.StepCollectionModel
	if err := json.Unmarshal(m.spec, &stepLib); err != nil {
		return nil, err
	}

	return validateTags(stepLib)
}

func toErrors(valErrs []jsonschema.ValError) []error {
	var errs []error
	for _, valErr := range valErrs {
		errs = append(errs, errors.New(valErr.Error()))
	}
	return errs
}

var projectTypeTags = map[string]bool{
	"ios":          true,
	"macos":        true,
	"android":      true,
	"xamarin":      true,
	"react-native": true,
	"cordova":      true,
	"ionic":        true,
	"flutter":      true,
	"web":          true,
}

var typeTags = map[string]bool{
	"access-control": true,
	"artifact-info":  true,
	"installer":      true,
	"deploy":         true,
	"utility":        true,
	"dependency":     true,
	"code-sign":      true,
	"build":          true,
	"test":           true,
	"notification":   true,
}

func validateTags(stepLib models.StepCollectionModel) ([]error, error) {
	var errs []error
	for stepID, stepGroup := range stepLib.Steps {
		var version string
		var step models.StepModel

		if stepGroup.LatestVersionNumber != "" {
			version = stepGroup.LatestVersionNumber

			var ok bool
			step, ok = stepGroup.Versions[version]
			if !ok {
				return nil, fmt.Errorf("%s latest version (%s) not found", stepID, version)
			}
		} else {
			for v, s := range stepGroup.Versions {
				version = v
				step = s
				break
			}
		}

		if len(step.TypeTags) == 0 {
			errs = append(errs, fmt.Errorf("%s (version %s) has no type_tags", stepID, version))
		} else {
			for _, tag := range step.TypeTags {
				if _, ok := typeTags[tag]; !ok {
					errs = append(errs, fmt.Errorf("%s (version %s) has invalid type_tag: %s", stepID, version, tag))
				}
			}
		}

		for _, tag := range step.ProjectTypeTags {
			if _, ok := projectTypeTags[tag]; !ok {
				errs = append(errs, fmt.Errorf("%s (version %s) has invalid project_type_tag: %s", stepID, version, tag))
			}
		}
	}

	return errs, nil
}
