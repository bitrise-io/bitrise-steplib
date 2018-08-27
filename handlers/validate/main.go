package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/validators/valueoptions"
	"github.com/bitrise-io/go-utils/log"
)

// Validator runner interface
type Validator interface {
	Validate() error
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
	log.Infof("Running validations:")
	validators := []Validator{
		&valueoptions.Validator{},
	}

	for i, validator := range validators {
		log.Printf("- (%d/%d) Running: %s", i+1, len(validators), validator)
		if err := validator.Validate(); err != nil {
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
