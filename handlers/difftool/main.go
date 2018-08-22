package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/bitrise-tools/go-steputils/stepconf"
	version "github.com/hashicorp/go-version"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
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

func auditChangedSteps(collectionURI, stepID, stepVer string) error {
	stepMainDirPth := "./steps/" + stepID
	versionDirs, err := ioutil.ReadDir(stepMainDirPth)
	if err != nil {
		failf("Failed to list dir, error: %s", err)
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

		stepPrevTmpDir, err := pathutil.NormalizedOSTempDirPath("_prev_")
		if err != nil {
			return err
		}

		currentTmpDir, err := pathutil.NormalizedOSTempDirPath("_current_")
		if err != nil {
			return err
		}

		cmd := command.New("stepman", "activate", "--collection", collectionURI, "--id", stepID, "--version", prevVersion, "--path", stepPrevTmpDir)
		fmt.Println()
		log.Donef("$ %s", cmd.PrintableCommandArgs())
		fmt.Println()

		if err := cmd.SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
			return err
		}

		cmd = command.New("stepman", "activate", "--collection", collectionURI, "--id", stepID, "--version", stepVer, "--path", currentTmpDir)
		fmt.Println()
		log.Donef("$ %s", cmd.PrintableCommandArgs())
		fmt.Println()

		if err := cmd.SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
			return err
		}

		// cmd := command.New(
		// 	git difftool -y -x "colordiff --suppress-common-lines  -y -W 250" 0.9.0 0.9.1
		// )
		// fmt.Println()
		// log.Donef("$ %s", cmd.PrintableCommandArgs())
		// fmt.Println()

		// diffOut, _ := cmd.RunAndReturnTrimmedCombinedOutput()

		fmt.Println()
		fmt.Println("========== DIFF ====================")
		//
		////
		//
		fmt.Println("====================================")
	} else {
		log.Warnf("FIRST VERSION - can't diff against previous version")
	}

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

	if err := auditChangedSteps(c.CollectionURI, "android-build", "0.9.4"); err != nil {
		panic(err)
	}
}
