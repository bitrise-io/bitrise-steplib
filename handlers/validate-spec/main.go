package main

import (
	"encoding/json"
	"errors"
	"os"

	rice "github.com/GeertJohan/go.rice"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/qri-io/jsonschema"
)

// Config ...
type Config struct {
	Spec bool `env:"spec"`
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
}

func loadAsset(name string) ([]byte, error) {
	assetsBox, err := rice.FindBox("assets")
	if err != nil {
		return nil, err
	}

	return assetsBox.Bytes(name)
}

func validateSchema(spec []byte) error {
	rs := &jsonschema.RootSchema{}
	if err := json.Unmarshal(schemaData, rs); err != nil {
		panic("unmarshal schema: " + err.Error())
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

var schemaData []byte

func init() {
	var err error
	schemaData, err = loadAsset("schema.json")
	if err != nil {
		panic(err)
	}
}
