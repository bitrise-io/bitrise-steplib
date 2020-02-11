package main

import (
	"encoding/json"
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

func rebuildAPICall() (string, error) {
	buildParamToEnvVar := map[string]string{
		"workflow_id":                 "BITRISE_TRIGGERED_WORKFLOW_ID",
		"commit_hash":                 "BITRISE_GIT_COMMIT",
		"branch":                      "BITRISE_GIT_BRANCH",
		"branch_dest":                 "BITRISEIO_GIT_BRANCH_DEST",
		"pull_request_id":             "PULL_REQUEST_ID",
		"pull_request_repository_url": "BITRISEIO_PULL_REQUEST_REPOSITORY_URL",
		"pull_request_merge_branch":   "BITRISEIO_PULL_REQUEST_MERGE_BRANCH",
	}
	buildParamToContents := map[string]string{
		"BYPASS_GATE":    "true",
		"commit_message": "Rebuilding with manually accepted step-info change.",
	}
	for buildParam, envName := range buildParamToEnvVar {
		buildParamToContents[buildParam] = os.Getenv(strings.ToUpper(envName))
	}

	buildParams, err := json.Marshal(buildParamToContents)
	if err != nil {
		return "", fmt.Errorf("rebuildAPICall: failed to marshal build params, envVars: %s, %v", buildParamToEnvVar, err)
	}

	return fmt.Sprintf(`$ curl https://app.bitrise.io/app/a0bac497f75e1490/build/start.json --data '{"hook_info":{"type":"bitrise","build_trigger_token":" < insert_build_trigger_token > "},
"build_params": %s,
"triggered_by":"curl"}'`, string(buildParams)), nil
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

	if !c.BypassGate {
		for _, file := range changedFiles {
			if strings.HasSuffix(file, "step-info.yml") {
				rebuildCall, err := rebuildAPICall()
				if err != nil {
					log.Warnf("%s", err)
				}
				failf("step-info.yml has changed, please re-check changes and rebuild with BYPASS_GATE env to true to pass this check, using: " + rebuildCall)
			}
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
