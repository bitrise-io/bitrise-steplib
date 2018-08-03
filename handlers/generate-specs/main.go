package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/bitrise-tools/go-steputils/tools"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

const (
	envSpecKey     = "SPEC_JSON"
	envSlimSpecKey = "SLIM_SPEC_JSON"
)

// Config ...
type Config struct {
	IsCI          bool   `env:"CI"`
	CollectionURI string `env:"collection_uri,required"`
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

	tmpPath, err := pathutil.NormalizedOSTempDirPath("collections")
	if err != nil {
		failf("Failed to get temp path, error: %s", err)
	}

	fmt.Println()

	log.Infof("Clean collection:")
	if err := command.New("stepman", "delete", "-c", c.CollectionURI).SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
		failf("Failed to run stepman delete, error: %s", err)
	}
	log.Donef("- Done")

	fmt.Println()

	log.Infof("Generating specs:")
	if err := command.New("stepman", "export-spec", "--steplib", c.CollectionURI, "--output", filepath.Join(tmpPath, "spec.json"), "--export-type", "full").SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
		failf("Failed to run stepman delete, error: %s", err)
	}
	log.Printf("- spec.json")

	if err := command.New("stepman", "export-spec", "--steplib", c.CollectionURI, "--output", filepath.Join(tmpPath, "slim-spec.json"), "--export-type", "latest").SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
		failf("Failed to run stepman delete, error: %s", err)
	}
	log.Printf("- slim-spec.json")
	log.Donef("- Done")

	fmt.Println()

	log.Infof("Exporting envs:")
	if err := tools.ExportEnvironmentWithEnvman(envSpecKey, filepath.Join(tmpPath, "spec.json")); err != nil {
		failf("Failed to export environment variable: %s", envSpecKey)
	}
	log.Printf("  Env    [ $%s = %s ]", envSpecKey, filepath.Join(tmpPath, "spec.json"))
	if err := tools.ExportEnvironmentWithEnvman(envSlimSpecKey, filepath.Join(tmpPath, "slim-spec.json")); err != nil {
		failf("Failed to export environment variable: %s", envSlimSpecKey)
	}
	log.Printf("  Env    [ $%s = %s ]", envSlimSpecKey, filepath.Join(tmpPath, "slim-spec.json"))
}
