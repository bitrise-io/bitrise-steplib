package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/bitrise-tools/go-steputils/tools"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

const (
	envIDKey        = "STEP_INFO_ID"
	envVersionKey   = "STEP_INFO_VERSION"
	envCompositeKey = "STEP_INFO_COMPOSITE"
)

// Config ...
type Config struct {
	IsCI bool `env:"CI"`
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

	fmt.Println()
	log.Infof("Looking for file changes:")
	cmdArgs := []string{"diff", "--name-only"}
	if c.IsCI {
		cmdArgs = append(cmdArgs, "HEAD", "origin/master")
	}

	output, err := command.New("git", cmdArgs...).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		failf("Diff failed, output: %s, error: %s", output, err)
	}

	log.Printf(output)
	log.Donef("- Done")
	fmt.Println()

	log.Infof("Find step change:")
	version, id := "", ""
	found := 0
	for _, file := range strings.Split(output, "\n") {
		if strings.HasSuffix(file, "step.yml") && strings.HasPrefix(file, "steps") {
			parent := filepath.Dir(file)
			version = filepath.Base(parent)
			parent = filepath.Dir(parent)
			id = filepath.Base(parent)
			log.Printf("%s - %s@%s", file, id, version)
			found++
		}
	}
	if found > 1 {
		failf("Multiple step.yml change, skip finding step-info")
	}
	if version == "" && id == "" {
		failf("No step change found.")
	}
	log.Donef("- Done")
	fmt.Println()

	log.Infof("Exporting envs:")
	if err := tools.ExportEnvironmentWithEnvman(envIDKey, id); err != nil {
		failf("Failed to export environment variable: %s", envIDKey)
	}
	log.Printf("  Env    [ $%s = %s ]", envIDKey, id)
	if err := tools.ExportEnvironmentWithEnvman(envVersionKey, version); err != nil {
		failf("Failed to export environment variable: %s", envVersionKey)
	}
	log.Printf("  Env    [ $%s = %s ]", envVersionKey, version)
	if err := tools.ExportEnvironmentWithEnvman(envCompositeKey, id+"@"+version); err != nil {
		failf("Failed to export environment variable: %s", envCompositeKey)
	}
	log.Printf("  Env    [ $%s = %s ]", envCompositeKey, id+"@"+version)
}
