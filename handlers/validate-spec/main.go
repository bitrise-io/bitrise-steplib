package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	rice "github.com/GeertJohan/go.rice"
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

	if err := validate(c.SpecPth); err != nil {
		failf("Spec (%s) validation failed: %s", err)
	}
}

func validate(pth string) error {
	spec, err := fileutil.ReadBytesFromFile(pth)
	if err != nil {
		return err
	}

	schema, err := loadAsset("schema.json")
	if err != nil {
		panic(err)
	}

	return validateSchema(spec, schema)
}

func loadAsset(name string) ([]byte, error) {
	assetsPth := filepath.Join(os.Getenv("BITRISE_STEP_SOURCE_DIR"), "assets")
	fmt.Printf("assetsPth: %s\n", assetsPth)
	assetsBox, err := rice.FindBox(assetsPth)
	if err != nil {
		return nil, err
	}

	return assetsBox.Bytes(name)
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
		for _, e := range issues {
			errs += e.Error()
			errs += "\n"
		}
		return errors.New(errs)
	}

	return nil
}
