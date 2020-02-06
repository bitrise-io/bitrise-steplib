package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

const (
	baseBranch = "master"
)

// Config ...
type Config struct {
	BypassGate bool `env:"BYPASS_GATE"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	var c Config
	if err := stepconf.Parse(&c); err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(c)
	fmt.Println()

	log.Infof("Checking changed files:")

	changedFiles, err := changedFiles(baseBranch)
	if err != nil {
		failf("Failed to get changed files for branch: %s: %s", baseBranch, err)
	}

	for _, file := range changedFiles {
		if strings.HasSuffix(file, "step-info.yml") && !c.BypassGate {
			//fail here
		}
	}

	fmt.Println()
	log.Donef("- Done")
}

func changedFiles(branch string) ([]string, error) {
	cmd := command.New("git", "diff", "--name-only", "HEAD", "origin/"+branch)
	output, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff command returned error: %s: %s", cmd.PrintableCommandArgs(), err)
	}

	var files []string
	for _, fileLine := range strings.Split(output, "\n") {
		if file := strings.TrimSpace(fileLine); len(file) > 0 {
			files = append(files, file)
		}
	}

	return files, nil
}
