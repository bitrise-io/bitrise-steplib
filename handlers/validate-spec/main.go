package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate-spec/validator"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

// Config ...
type Config struct {
	SpecPth   string `env:"spec_path,file"`
	SchemaPth string `env:"schema_path,file"`
}

func main() {
	var c Config
	err := stepconf.Parse(&c)
	if err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	specErrs, err := validate(c.SpecPth, c.SchemaPth)
	if err != nil {
		failf(err.Error())
	}

	if err := joinErrors(specErrs); err != nil {
		failf("Invalid spec\n%s", err)
	}

	log.Printf("Spec and slim spec are valid")
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func joinErrors(errs []error) error {
	var s []string
	for _, err := range errs {
		s = append(s, err.Error())
	}

	if len(s) > 0 {
		return errors.New(strings.Join(s, "\n"))
	}
	return nil
}

func validate(specPth, schemaPth string) ([]error, error) {
	specReader, err := os.Open(specPth)
	if err != nil {
		return nil, fmt.Errorf("failed to open spec: %s", err)
	}

	schemaReader, err := os.Open(schemaPth)
	if err != nil {
		return nil, fmt.Errorf("failed to open schema: %s", err)
	}

	specErrs, err := validateSpec(specReader, schemaReader)
	if err != nil {
		return nil, fmt.Errorf("failed to validate spec: %s", err)
	}

	return specErrs, nil
}

func validateSpec(specReader, schemaReader io.Reader) ([]error, error) {
	v, err := validator.New(specReader)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise validator: %s", err)
	}

	schemaErrs, err := v.ValidateSchema(schemaReader)
	if err != nil {
		return nil, fmt.Errorf("failed to validate schame: %s", err)
	}

	tagErrs, err := v.ValidateTags()
	if err != nil {
		return schemaErrs, fmt.Errorf("failed to validate type and project type tags: %s", err)
	}
	return append(schemaErrs, tagErrs...), nil
}
