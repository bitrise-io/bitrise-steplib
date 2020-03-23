package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/qri-io/jsonschema"
)

// Config ...
type Config struct {
	SpecPth string `env:"spec"`
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

	schemaPth := filepath.Join(os.Getenv("BITRISE_SOURCE_DIR"), "schema.json")
	if err := validate(c.SpecPth, schemaPth); err != nil {
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

	return validateSchema(spec, schema)
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
