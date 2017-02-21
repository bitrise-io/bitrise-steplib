package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/hashicorp/go-version"
)

var (
	collectionID    = ""
	isLocalTestMode = false
)

// -----------------------------------
// --- UTIL FUNCTIONS

func runCommandAndReturnCombinedOutputs(isDebug bool, name string, args ...string) (string, error) {
	cmd := command.New(name, args...)
	outStr, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if isDebug && err != nil {
		log.Errorf("Failed to run command: %#v", cmd)
	}
	return strings.TrimSpace(outStr), err
}

func detectStepIDAndVersionFromPath(pth string) (stepID, stepVersion string, err error) {
	pathComps := strings.Split(pth, "/")
	if len(pathComps) < 4 {
		err = fmt.Errorf("path should contain at least 4 components: steps, step-id, step-version, step.yml: %s", pth)
		return
	}
	// we only care about the last 4 component of the path
	pathComps = pathComps[len(pathComps)-4:]
	if pathComps[0] != "steps" {
		err = fmt.Errorf("invalid step.yml path, 'steps' should be included right before the step-id: %s", pth)
		return
	}
	if pathComps[3] != "step.yml" {
		err = fmt.Errorf("invalid step.yml path, should end with 'step.yml': %s", pth)
		return
	}
	stepID = pathComps[1]
	stepVersion = pathComps[2]
	return
}

func collectVersionsFromDir(dirPth string) ([]*version.Version, error) {
	dirInfos, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return []*version.Version{}, fmt.Errorf("failed to list dir: %s", err)
	}
	versions := []*version.Version{}
	for _, aDirInfo := range dirInfos {
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

func auditChangedStepInfoYML(stepInfoYmlPth string) error {
	type StepGroupInfoModel struct {
		RemovalDate    string            `json:"removal_date,omitempty" yaml:"removal_date,omitempty"`
		DeprecateNotes string            `json:"deprecate_notes,omitempty" yaml:"deprecate_notes,omitempty"`
		AssetURLs      map[string]string `json:"asset_urls,omitempty" yaml:"asset_urls,omitempty"`
	}

	bytes, err := fileutil.ReadBytesFromFile(stepInfoYmlPth)
	if err != nil {
		return fmt.Errorf("failed to read global step info (%s), error: %s", stepInfoYmlPth, err)
	}

	var stepGroupInfo StepGroupInfoModel
	if err := yaml.Unmarshal(bytes, &stepGroupInfo); err != nil {
		return fmt.Errorf("failed to parse global step info (%s), error: %s", stepInfoYmlPth, err)
	}

	return nil
}

func auditChangedStepYML(stepYmlPth string) error {
	log.Infof("Audit changed step.yml: ", stepYmlPth)

	stepID, stepVer, err := detectStepIDAndVersionFromPath(stepYmlPth)
	if err != nil {
		return fmt.Errorf("Audit failed for (%s), error: %s", stepYmlPth, err)
	}

	log.Printf("Step's main folder content:")

	stepMainDirPth := "./steps/" + stepID
	lsOut, err := runCommandAndReturnCombinedOutputs(true, "ls", "-alh", stepMainDirPth)
	if err != nil {
		return fmt.Errorf("failed to list the step's main folder (%s) content, output: %s, error: %s", stepMainDirPth, lsOut, err)
	}

	fmt.Println()

	//
	versions, err := collectVersionsFromDir(stepMainDirPth)
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

		log.Warnf("Diff step: %s | %s <-> %s", stepID, stepVer, prevVersion)

		diffOut, _ := runCommandAndReturnCombinedOutputs(
			false,
			"diff",
			path.Join(stepMainDirPth, prevVersion, "step.yml"),
			path.Join(stepMainDirPth, stepVer, "step.yml"),
		)

		fmt.Println()
		fmt.Println()
		fmt.Println("========== DIFF ====================")
		fmt.Println(diffOut)
		fmt.Println("====================================")
		fmt.Println()
		fmt.Println()
	} else {
		log.Warnf("FIRST VERSION - can't diff against previous version")
	}

	log.Infof("Auditing step: %s | version: %s", stepID, stepVer)
	//
	tmpStepActPth, err := pathutil.NormalizedOSTempDirPath(stepID + "--" + stepVer)
	if err != nil {
		return fmt.Errorf("failed to create tmp dir, error: %s", err)
	}
	//
	output, err := runCommandAndReturnCombinedOutputs(true,
		"stepman", "activate",
		"--collection", collectionID,
		"--id", stepID,
		"--version", stepVer,
		"--path", tmpStepActPth,
	)
	if err != nil {
		return fmt.Errorf("failed to run stepman activate, output: %s, error: %s", output, err)
	}

	log.Printf("stepman activate output: %s", output)
	log.Donef("SUCCESSFUL audit")

	return nil
}

func fatalf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

// -----------------------------------
// --- MAIN

func init() {
	flag.StringVar(&collectionID, "collectionid", "", "Collection ID to use")
	flag.BoolVar(&isLocalTestMode, "localtest", false, "Enable local test mode - runs `git diff` on local changes instead of HEAD..origin/master")
}

func main() {
	fmt.Println()

	// --- INPUTS
	flag.Parse()
	if collectionID == "" {
		fatalf("Collection ID not provided!")
	}

	// --- MAIN
	log.Infof("Auditing changed steps...")

	log.Printf("git fetch...")

	if output, err := runCommandAndReturnCombinedOutputs(true, "git", "fetch"); err != nil {
		fatalf("git fecth failed, output: %s, error: %s", output, err)
	}

	log.Printf("git diff...")

	diffOutput := ""
	var diffErr error
	//
	if isLocalTestMode {
		diffOutput, diffErr = runCommandAndReturnCombinedOutputs(true, "git", "diff", "--name-only", "--cached", "upstream/master")
	} else {
		diffOutput, diffErr = runCommandAndReturnCombinedOutputs(true, "git", "diff", "--name-only", "HEAD", "origin/master")
	}

	if diffErr != nil {
		fatalf("git diff failed, output: %s, error: %s", diffOutput, diffErr)
	}

	changedFilePaths := strings.Split(diffOutput, "\n")

	log.Printf("Changed files:")
	for _, pth := range changedFilePaths {
		log.Printf("- %s", pth)
	}

	for _, aPth := range changedFilePaths {
		if strings.HasSuffix(aPth, "step.yml") {
			if isExist, err := pathutil.IsPathExists(aPth); err != nil {
				fatalf("Failed to check if path (%s) exists, error: %s", aPth, err)
			} else if !isExist {
				fatalf("step.yml was removed: %s", aPth)
			}

			if err := auditChangedStepYML(aPth); err != nil {
				fatalf("Failed to audit step (%s), err: %s", aPth, err)
			}
		} else if strings.HasSuffix(aPth, "step-info.yml") {
			if isExist, err := pathutil.IsPathExists(aPth); err != nil {
				fatalf("Failed to check if path (%s) exists, error: %s", aPth, err)
			} else if !isExist {
				fatalf("step-info.yml was removed: %s", aPth)
			}

			if err := auditChangedStepInfoYML(aPth); err != nil {
				fatalf("Failed to audit global step info (%s), err: %s", aPth, err)
			}
		} else if dir := filepath.Dir(aPth); strings.HasSuffix(dir, "assets") {
			log.Warnf("asset, skipping audit: %s", aPth)
		} else {
			log.Warnf("Unkown file, skipping audit: %s", aPth)
		}
	}

	log.Donef("DONE")
}
