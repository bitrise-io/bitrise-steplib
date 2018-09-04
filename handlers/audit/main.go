package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/bitrise-tools/go-steputils/stepconf"
	version "github.com/hashicorp/go-version"

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
	IsCI bool   `env:"CI"`
	Step string `env:"step"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func checkIcon(stepID string) {
	activeExtensions := []string{"png", "svg"}

	log.Printf("Checking icon for: %s", stepID)
	hasIcon := false
	for _, ext := range activeExtensions {
		iconPth := fmt.Sprintf("steps/%s/assets/icon.%s", stepID, ext)
		if isExist, err := pathutil.IsPathExists(iconPth); err != nil {
			failf("Failed to check if path (%s) exists, error: %s", iconPth, err)
		} else if isExist {
			log.Donef("- %s", iconPth)
			hasIcon = true
		} else {
			log.Printf("- %s", iconPth)
		}
	}
	if hasIcon {
		log.Donef("> Icon found!")
		fmt.Println()
	} else {
		log.Errorf("> No icon found")
		fmt.Println()
	}
}

func collectVersionsFromDir(versionDirs []os.FileInfo) ([]*version.Version, error) {
	versions := []*version.Version{}
	for _, aDirInfo := range versionDirs {
		aVerStr := aDirInfo.Name()
		if aVerStr == "assets" {
			continue
		}
		if aVerStr == "step-info.yml" {
			continue
		}

		ver, err := version.NewVersion(aVerStr)
		if err != nil {
			return []*version.Version{}, fmt.Errorf("failed to create version from string: %s | error: %s", aVerStr, err)
		}
		versions = append(versions, ver)
	}
	return versions, nil
}

func auditChangedStepYML(stepID, stepVer string) error {

	log.Printf("Step's main folder content:")

	stepMainDirPth := "./steps/" + stepID
	versionDirs, err := ioutil.ReadDir(stepMainDirPth)
	if err != nil {
		failf("Failed to list dir, error: %s", err)
	}
	for _, f := range versionDirs {
		if !f.IsDir() {
			continue
		}
		log.Printf(f.Name())
	}

	versions, err := collectVersionsFromDir(versionDirs)
	if err != nil {
		return fmt.Errorf("failed to collect versions, error: %s", err)
	}
	if len(versions) > 1 {
		sort.Sort(version.Collection(versions))
		prevVersion := ""
		for _, aVer := range versions {
			if aVer.String() == stepVer {
				// stop
				break
			}
			prevVersion = aVer.String()
		}

		fmt.Println()
		log.Warnf("Diff step: %s | %s >> %s", stepID, prevVersion, stepVer)

		cmd := command.New(
			"diff",
			path.Join(stepMainDirPth, prevVersion, "step.yml"),
			path.Join(stepMainDirPth, stepVer, "step.yml"),
		)
		fmt.Println()
		log.Donef("$ %s", cmd.PrintableCommandArgs())
		fmt.Println()

		diffOut, _ := cmd.RunAndReturnTrimmedCombinedOutput()

		fmt.Println()
		fmt.Println("========== DIFF ====================")
		fmt.Println(diffOut)
		fmt.Println("====================================")
	} else {
		log.Warnf("FIRST VERSION - can't diff against previous version")
	}

	fmt.Println()
	log.Infof("Auditing step: %s | version: %s", stepID, stepVer)

	tmpStepActPth, err := pathutil.NormalizedOSTempDirPath(stepID + "--" + stepVer)
	if err != nil {
		return fmt.Errorf("failed to create tmp dir, error: %s", err)
	}

	cmd := command.New(
		"stepman", "activate",
		"--collection", "file://./",
		"--id", stepID,
		"--version", stepVer,
		"--path", tmpStepActPth,
	)
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())
	fmt.Println()

	output, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run stepman activate, output: %s, error: %s", output, err)
	}

	fmt.Println()
	log.Printf("stepman activate output:")
	log.Printf(output)

	fmt.Println()
	log.Donef("SUCCESSFUL audit")

	return nil
}

func main() {
	var c Config
	err := stepconf.Parse(&c)
	if err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(c)
	fmt.Println()

	fullAudit := c.Step == ""
	stepID := ""
	stepVer := ""
	if !fullAudit {
		s := strings.Split(c.Step, "@")
		if len(s) != 2 {
			failf("invalid step composite")
		}
		stepID = s[0]
		stepVer = s[1]
	}

	log.Infof("Clean collection:")

	cmd := command.New("stepman", "delete", "-c", "file://./")
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())
	fmt.Println()

	if err := cmd.SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
		failf("Failed to run stepman delete, error: %s", err)
	}

	cmd = command.New("stepman", "setup", "-c", "file://./")
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())
	fmt.Println()

	if err := cmd.SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
		failf("Failed to run stepman delete, error: %s", err)
	}
	log.Donef("- Done")
	fmt.Println()

	if fullAudit {
		log.Infof("Checking icons:")
		stepIDs, err := ioutil.ReadDir("./steps")
		if err != nil {
			failf("Failed to list dir, error: %s", err)
		}
		for _, f := range stepIDs {
			if !f.IsDir() {
				continue
			}
			checkIcon(f.Name())
		}
		log.Infof("Auditing steps:")

		cmd = command.New("stepman", "audit", "-c", "file://./")
		fmt.Println()
		log.Donef("$ %s", cmd.PrintableCommandArgs())
		fmt.Println()

		if err := cmd.SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
			failf("Failed to run stepman delete, error: %s", err)
		}
	} else {
		log.Infof("Checking icon:")
		checkIcon(stepID)
		fmt.Println()

		log.Infof("Auditing step:")
		if err := auditChangedStepYML(stepID, stepVer); err != nil {
			failf("Audit failed, error: %s", err)
		}
	}
}
