package collection

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/hashicorp/go-version"

	"github.com/bitrise-io/bitrise-steplib/handlers/validate/steplib"
	"github.com/bitrise-io/go-utils/command"
)

const testBitriseYML = `format_version: "4"
default_step_lib_source: file://./

workflows:
  test:
    steps:
    - script:
        inputs:
        - content: echo "successful"
`

// Validator is matching for the validator interface
type Validator struct{}

// IsSkippable sets the validation task to skip failures
func (v *Validator) IsSkippable() bool {
	return false
}

func getTestableCLIVersionDownloadURLs() ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/bitrise-io/bitrise/releases?per_page=20", nil)

	if token := os.Getenv("github_access_token"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Failed to close http response body, error: %s", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		fmt.Println("body:", string(body))
		return nil, err
	}

	os := "Darwin"
	arch := "x86_64"

	if runtime.GOOS != "darwin" {
		os = "Linux"
	}

	// before version 1.21.0 we had a validator that fails for an unknown input option field and here we
	// implemented the new is_sensitive input option which then causes older versions to fail with the current StepLib
	latestSupportedVersion, err := version.NewVersion("1.20.0")
	if err != nil {
		return nil, err
	}

	var releaseTags []string
	for _, release := range releases {
		if version, err := version.NewVersion(release.TagName); err == nil && version.GreaterThan(latestSupportedVersion) && !isUnsupportedVersion(version, "1.39.0", "1.39.1") {
			releaseTags = append(releaseTags, fmt.Sprintf("https://github.com/bitrise-io/bitrise/releases/download/%s/bitrise-%s-%s", release.TagName, os, arch))
		}
	}

	return releaseTags, nil
}

func isUnsupportedVersion(ve *version.Version, unsupportedVersions ...string) bool {
	for _, versionStr := range unsupportedVersions {
		if ve.String() == versionStr {
			return true
		}
	}
	return false
}

func setupBinary(url string) (string, error) {
	tmpPath, err := pathutil.NormalizedOSTempDirPath("cli_version_test")
	if err != nil {
		return "", err
	}

	tmpBinaryPath := filepath.Join(tmpPath, "bitrise")

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Failed to close http response body, error: %s", err)
		}
	}()

	binaryFile, err := os.Create(tmpBinaryPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := binaryFile.Close(); err != nil {
			log.Errorf("Failed to close file, error: %s", err)
		}
	}()

	if _, err := io.Copy(binaryFile, resp.Body); err != nil {
		return "", err
	}

	if err := os.Chmod(tmpBinaryPath, 0777); err != nil {
		return "", err
	}

	return tmpBinaryPath, nil
}

// Validate is the logic handler of the task
func (v *Validator) Validate(sl steplib.StepLib) error {
	cliVersionURLs, err := getTestableCLIVersionDownloadURLs()
	if err != nil {
		return err
	}

	for _, cliVersionURL := range cliVersionURLs {
		fmt.Println(" - downloading and running:", cliVersionURL)

		tmpBinaryPath, err := setupBinary(cliVersionURL)
		if err != nil {
			return err
		}

		if err := os.RemoveAll(filepath.Join(os.Getenv("HOME"), ".bitrise/tools")); err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(os.Getenv("HOME"), ".stepman")); err != nil {
			return err
		}

		out, err := command.New(tmpBinaryPath, "version").RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			return fmt.Errorf(" - CLI run failed, output:\n%s\n\nerror: %s", out, err)
		}
		fmt.Println("   > version:", out)

		out, err = command.New(tmpBinaryPath, "setup").RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			return fmt.Errorf(" - CLI run failed, output:\n%s\n\nerror: %s", out, err)
		}

		out, err = command.New(tmpBinaryPath, "run", "--config-base64", base64.StdEncoding.EncodeToString([]byte(testBitriseYML)), "test").RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			return fmt.Errorf(" - CLI run failed, output:\n%s\n\nerror: %s", out, err)
		}
	}
	return nil
}

// String will return a short summary of the validator task
func (v *Validator) String() string {
	return "Ensure CLI is working properly with the new collection"
}
