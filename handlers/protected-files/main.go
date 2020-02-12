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
	baseBranch             = "master"
	PRCheckWorkflow        = "pr_check"
	changesApprovedEnvName = "STEPLIB_CHANGES_APPROVED"
)

// Config ...
type Config struct {
	IsApproved bool `env:"changes_approved"`
}

// BuildTriggerRequestModel ...
type BuildTriggerRequestModel struct {
	HookInfo    HookInfoModel    `json:"hook_info"`
	BuildParams BuildParamsModel `json:"build_params"`
	TriggeredBy string           `json:"triggered_by,omitempty"`
}

// HookInfoModel ...
type HookInfoModel struct {
	Type              string `json:"type"`
	BuildTriggerToken string `json:"build_trigger_token,omitempty"`
	APIToken          string `json:"api_token,omitempty"`
}

// BuildParamsModel ...
type BuildParamsModel struct {
	Branch                   string                     `json:"branch"`
	Tag                      string                     `json:"tag"`
	CommitHash               string                     `json:"commit_hash"`
	CommitMessage            string                     `json:"commit_message"`
	WorkflowID               string                     `json:"workflow_id"`
	BranchDest               string                     `json:"branch_dest"`
	PullRequestID            string                     `json:"pull_request_id"`
	PullRequestRepositoryURL string                     `json:"pull_request_repository_url"`
	PullRequestMergeBranch   string                     `json:"pull_request_merge_branch"`
	PullRequestHeadBranch    string                     `json:"pull_request_head_branch"`
	Environments             []EnvironmentVariableModel `json:"environments"`
	BranchDestRepoOwner      string                     `json:"branch_dest_repo_owner"`
	BranchRepoOwner          string                     `json:"branch_repo_owner"`
}

// EnvironmentVariableModel ...
type EnvironmentVariableModel struct {
	MappedTo string `json:"mapped_to"`
	Value    string `json:"value"`
	IsExpand bool   `json:"is_expand"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func rebuildInstructions() (string, error) {
	const approvedByEnv = "APPROVED_BY"
	const buildTriggerEnv = "BUILD_TRIGGER_TOKEN"
	buildParams := BuildParamsModel{
		Branch:                   os.Getenv("BITRISE_GIT_BRANCH"),
		Tag:                      os.Getenv("BITRISE_GIT_TAG"),
		CommitHash:               os.Getenv("BITRISE_GIT_COMMIT"),
		CommitMessage:            "Rebuilding with manually accepted step-info change. Approved by: $" + approvedByEnv,
		WorkflowID:               PRCheckWorkflow,
		BranchDest:               os.Getenv("BITRISEIO_GIT_BRANCH_DEST"),
		PullRequestID:            os.Getenv("PULL_REQUEST_ID"),
		PullRequestRepositoryURL: os.Getenv("BITRISEIO_PULL_REQUEST_REPOSITORY_URL"),
		PullRequestMergeBranch:   os.Getenv("BITRISEIO_PULL_REQUEST_MERGE_BRANCH"),
		PullRequestHeadBranch:    os.Getenv("BITRISEIO_PULL_REQUEST_HEAD_BRANCH"),
		Environments: []EnvironmentVariableModel{
			EnvironmentVariableModel{
				MappedTo: changesApprovedEnvName,
				Value:    "true",
			},
		},
	}
	buildTrigger := BuildTriggerRequestModel{
		HookInfo: HookInfoModel{
			Type:              "bitrise",
			BuildTriggerToken: "$" + buildTriggerEnv,
		},
		TriggeredBy: "curl",
		BuildParams: buildParams,
	}
	buildTriggerRequest, err := json.Marshal(buildTrigger)
	if err != nil {
		return "", fmt.Errorf("rebuildAPICall: failed to marshal build trigger request body, buildTrigger: %+v, %v", buildTrigger, err)
	}
	escapedBuildTriggerRequest := strings.ReplaceAll(string(buildTriggerRequest), "\"", "\\\"")
	buildStartURL := fmt.Sprintf("https://app.bitrise.io/app/%s/build/start.json", os.Getenv("BITRISE_APP_SLUG"))
	curlCommand := fmt.Sprintf("curl %s --data \"%s\"", buildStartURL, escapedBuildTriggerRequest)
	curlCommandWithParams := fmt.Sprintf("%s=bitbot; %s=token; %s", approvedByEnv, buildTriggerEnv, curlCommand)

	buildTriggerURL := fmt.Sprintf("Get build trigger URL here: https://app.bitrise.io/app/%s#/code.", os.Getenv("BITRISE_APP_SLUG"))

	return fmt.Sprintf("step-info.yml has changed, please re-check changes and rebuild. %s Command to rebuild: \n$ %s", buildTriggerURL, curlCommandWithParams), nil
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

	if !c.IsApproved {
		for _, file := range changedFiles {
			if strings.HasSuffix(file, "step-info.yml") {
				rebuildInstructions, err := rebuildInstructions()
				if err != nil {
					log.Warnf("%s", err)
				}
				failf(rebuildInstructions)
			}
		}
	}

	fmt.Println()
	log.Donef("- Done")
}
