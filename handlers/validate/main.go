package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/steplib"
	stepid "github.com/bitrise-io/bitrise-steplib/handlers/validate/validators/changed-stepid"
	"github.com/bitrise-io/bitrise-steplib/handlers/validate/validators/typetags"
	"github.com/bitrise-io/bitrise-steplib/handlers/validate/validators/valueoptions"
	"github.com/bitrise-io/go-utils/log"
)

// Validator runner interface
type Validator interface {
	Validate(steplib.StepLib) error
	IsSkippable() bool
	String() string
}

// Config ...
type Config struct {
	Step string `env:"step"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	log.Infof("Parsing StepLib:")
	sl, err := steplib.NewStepLib("./")
	if err != nil {
		failf("Failed to parse StepLib, error: %s", err)
	}
	log.Donef("> Done")
	fmt.Println()

	log.Infof("Running validations:")
	validators := []Validator{
		//&collection.Validator{},
		&valueoptions.Validator{},
		&typetags.Validator{},
		&stepid.Validator{},
	}

	for i, validator := range validators {
		log.Printf("- (%d/%d) Running: %s", i+1, len(validators), validator)
		if err := validator.Validate(sl); err != nil {
			if !validator.IsSkippable() {
				log.Errorf(" > Failed, error:")
				log.Printf("%s", err)
				os.Exit(1)
			} else {
				log.Warnf(" > Skipped, error:")
				log.Printf("%s", err)
			}
		} else {
			log.Donef(" > Successful")
		}
		fmt.Println()
	}
}
