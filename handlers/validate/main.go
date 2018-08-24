package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/tasks/valueoptions"
	"github.com/bitrise-io/go-utils/log"
)

// Task runner interface
type Task interface {
	Work() error
	IsSkippable() bool
	String() string
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	log.Infof("Running validation tasks:")
	tasks := []Task{
		valueoptions.Task{},
	}

	for i, task := range tasks {
		log.Printf("- (%d/%d) Running: %s", i+1, len(tasks), task)
		if err := task.Work(); err != nil {
			if !task.IsSkippable() {
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
