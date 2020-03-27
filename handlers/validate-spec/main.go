package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/stepman/models"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/qri-io/jsonschema"
)

// Config ...
type Config struct {
	SpecPth   string `env:"spec_path,required"`
	SchemaPth string `env:"schema_path,required"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	var c Config
	err := stepconf.Parse(&c)
	if err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	if err := validate(c.SpecPth, c.SchemaPth); err != nil {
		failf("Spec (%s) validation failed: %s", err)
	}
}

func validate(specPth, schemaPth string) error {
	spec, err := fileutil.ReadBytesFromFile(specPth)
	if err != nil {
		return err
	}

	schema, err := fileutil.ReadBytesFromFile(schemaPth)
	if err != nil {
		return err
	}

	schemaErr := validateSchema(spec, schema)
	tagErr := validateTypeAndProjectTypeTags(spec)

	var errs []string
	if schemaErr != nil {
		errs = append(errs, schemaErr.Error())
	}
	if tagErr != nil {
		errs = append(errs, tagErr.Error())
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func validateSchema(spec, schema []byte) error {
	rs := &jsonschema.RootSchema{}
	if err := json.Unmarshal(schema, rs); err != nil {
		return err
	}

	issues, err := rs.ValidateBytes(spec)
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		var errs string
		for i, issue := range issues {
			errs += issue.Error()
			if i < len(issues)-1 {
				errs += "\n"
			}
		}
		return errors.New(errs)
	}

	return nil
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

func validateTypeAndProjectTypeTags(spec []byte) error {
	var stepLib models.StepCollectionModel
	if err := json.Unmarshal(spec, &stepLib); err != nil {
		return err
	}

	var errs string
	for stepID, stepGroup := range stepLib.Steps {
		step := stepGroup.Versions[stepGroup.LatestVersionNumber]

		if len(step.TypeTags) == 0 {
			errs += fmt.Sprintf("%s (version %s) has no type_tags\n", stepID, stepGroup.LatestVersionNumber)
		} else {
			for _, tag := range step.TypeTags {
				if _, ok := typeTags[tag]; !ok {
					errs += fmt.Sprintf("%s (version %s) has invalid type_tag: %s\n", stepID, stepGroup.LatestVersionNumber, tag)
				}
			}
		}

		for _, tag := range step.ProjectTypeTags {
			if _, ok := projectTypeTags[tag]; !ok {
				errs += fmt.Sprintf("%s (version %s) has invalid project_type_tag: %s\n", stepID, stepGroup.LatestVersionNumber, tag)
			}
		}
	}

	if errs != "" {
		return errors.New(errs)
	}
	return nil
}
