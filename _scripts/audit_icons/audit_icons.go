package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
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
	fmt.Println()

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

	activeExtensions := []string{"png", "svg"}
	changedFilePaths := strings.Split(diffOutput, "\n")
	hasMissingIcon := false

	fmt.Println()
	log.Printf("Checking icons:")

	for _, aPth := range changedFilePaths {
		if strings.HasSuffix(aPth, "step.yml") {
			hasIcon := false
			rootPth := filepath.Join(filepath.Dir(aPth), "..")
			assetsPth := filepath.Join(rootPth, "assets")

			for _, ext := range activeExtensions {
				iconPth := filepath.Join(assetsPth, fmt.Sprintf("icon.%s", ext))
				if isExist, err := pathutil.IsPathExists(iconPth); err != nil {
					fatalf("Failed to check if path (%s) exists, error: %s", iconPth, err)
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
				hasMissingIcon = true
			}
		}
	}
	if hasMissingIcon {
		log.Errorf("ERROR")
	} else {
		log.Donef("DONE")
	}
}
