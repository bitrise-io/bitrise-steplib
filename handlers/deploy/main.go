package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-tools/go-steputils/stepconf"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

const (
	stepsAssetsBasePathInBucket  = "steps"
	stepArchivesBasePathInBucket = "step-archives"
)

// Config ...
type Config struct {
	IsCI                       bool   `env:"CI"`
	AwsAccessKey               string `env:"AWS_ACCESS_KEY_ID,required"`
	AwsSecretKey               string `env:"AWS_SECRET_ACCESS_KEY,required"`
	S3Bucket                   string `env:"S3_UPLOAD_BUCKET,required"`
	StepsDirPath               string `env:"steps_dir,required"`
	CollectionURI              string `env:"collection_uri,required"`
	CollectionSpecJSONPath     string `env:"spec_json_path,required"`
	CollectionSlimSpecJSONPath string `env:"slim_spec_json_path,required"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func prepareTmpStepZIPArchiveFromGit(collectionURI, stepID, stepVersion string) (string, error) {
	stepVerTmpDir, err := ioutil.TempDir("", stepID+"."+stepVersion+"--src")
	if err != nil {
		return "", err
	}
	stepZipArchivePth, err := ioutil.TempDir("", stepID+"."+stepVersion+"--zip")
	if err != nil {
		return "", err
	}
	stepZipArchivePth += "/step.zip"
	fmt.Println("stepVerTmpDir: ", stepVerTmpDir)
	fmt.Println("stepZipArchivePth: ", stepZipArchivePth)

	cmd := command.New("stepman", "activate", "--collection", collectionURI, "--id", stepID, "--version", stepVersion, "--path", stepVerTmpDir)
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())
	fmt.Println()

	if err := cmd.SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
		return "", err
	}

	// A bit of cleanup
	if err := os.RemoveAll(stepVerTmpDir + "/.git"); err != nil {
		return "", err
	}
	if err := os.RemoveAll(stepVerTmpDir + "/.DS_Store"); err != nil {
		return "", err
	}

	// zip it up!

	cmd = command.New("zip", "-r", stepZipArchivePth, ".")
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())
	fmt.Println()

	if err := cmd.SetDir(stepVerTmpDir).
		SetStdout(os.Stdout).SetStderr(os.Stderr).Run(); err != nil {
		return "", err
	}

	return stepZipArchivePth, nil
}

func uploadFileToS3(s3Bucket, filePth, pathInBucket string, otherOpts ...string) error {
	upldPth := "s3://" + s3Bucket
	if !strings.HasPrefix(pathInBucket, "/") {
		upldPth += "/"
	}
	upldPth += pathInBucket

	fmt.Println("Uploading file: ", filePth, " | to: ", upldPth)

	cmd := command.New("aws", append([]string{"s3", "cp", filePth, upldPth, "--acl", "public-read"}, otherOpts...)...)
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())
	fmt.Println()

	if output, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		fmt.Println()
		fmt.Println()
		fmt.Println("==========> [!] FAILED to upload file")
		fmt.Println("=> OUTPUT:")
		log.Printf("%s\n", output)
		fmt.Println()
		fmt.Println()
		return err
	}
	return nil
}

func uploadStepsDir(s3Bucket, stepsDirPath string) error {
	s3UploadFullPath := "s3://" + s3Bucket + "/" + stepsAssetsBasePathInBucket + "/"
	upStepsDirPth := stepsDirPath
	if !strings.HasSuffix(upStepsDirPth, "/") {
		upStepsDirPth += "/"
	}
	fmt.Println("Uploading from: ", upStepsDirPth, " | to: ", s3UploadFullPath)

	cmd := command.New("aws", "s3", "sync", upStepsDirPth, s3UploadFullPath, "--acl", "public-read")
	fmt.Println()
	log.Donef("$ %s", cmd.PrintableCommandArgs())
	fmt.Println()

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func uploadStepArchiveZIPs(collectionURI, s3Bucket, stepsDirPath string) error {
	// s3UploadBasePath := "s3://" + s3Bucket + "/" + stepArchivesBasePathInBucket
	s3ZipCheckBasePath := "https://" + s3Bucket + ".s3.amazonaws.com/" + stepArchivesBasePathInBucket

	stepYmlPattern := stepsDirPath + "/*/*/step.yml"
	fmt.Println("stepYmlPattern: ", stepYmlPattern)
	stepYmlPaths, err := filepath.Glob(stepYmlPattern)
	if err != nil {
		return err
	}
	fmt.Println("Found step count: ", len(stepYmlPaths))
	for idx, aStepYmlPth := range stepYmlPaths {
		fmt.Println()
		log.Printf("---> %d: %s", idx, aStepYmlPth)
		pathComps := strings.Split(aStepYmlPth, "/")
		if len(pathComps) < 4 {
			fmt.Println(" [!] The step.yml path should have at least 4 path components (steps, step-id, step-version, step.yml): ", aStepYmlPth)
			return fmt.Errorf("Invalid step.yml path: %s", aStepYmlPth)
		}
		pathComps = pathComps[len(pathComps)-4:]
		if pathComps[0] != "steps" {
			fmt.Println(" [!] Invalid step.yml path, 'steps' should be included right before the step-id: ", aStepYmlPth)
			return fmt.Errorf("Invalid step.yml path: %s", aStepYmlPth)
		}
		if pathComps[3] != "step.yml" {
			fmt.Println(" [!] Invalid step.yml path, should end with 'step.yml': ", aStepYmlPth)
			return fmt.Errorf("Invalid step.yml path: %s", aStepYmlPth)
		}
		stepID := pathComps[1]
		stepVersion := pathComps[2]

		// check if ZIP for this step&version already exists
		stepZipCheckURL := s3ZipCheckBasePath + "/" + stepID + "/" + stepVersion + "/step.zip"
		fmt.Println("Checking: ", stepZipCheckURL)
		resp, err := http.Head(stepZipCheckURL)
		if err != nil {
			return err
		}
		if resp.StatusCode == 200 {
			fmt.Println("---> zip found [OK]")
			continue
		}

		fmt.Println("zip NOT found or not accessible - creating and uploading...")

		zipPth, err := prepareTmpStepZIPArchiveFromGit(collectionURI, stepID, stepVersion)
		if err != nil {
			return err
		}
		fmt.Println("zipPth: ", zipPth)
		zipPthInBucket := stepArchivesBasePathInBucket + "/" + stepID + "/" + stepVersion + "/step.zip"
		if err := uploadFileToS3(s3Bucket, zipPth, zipPthInBucket); err != nil {
			return err
		}
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

	fmt.Println("==> Uploading steps/ dir...")
	fmt.Println()
	if err := uploadStepsDir(c.S3Bucket, c.StepsDirPath); err != nil {
		failf("Error: ", err)
	}

	fmt.Println()
	fmt.Println("==> Uploading step ZIPs...")
	fmt.Println()
	if err := uploadStepArchiveZIPs(c.CollectionURI, c.S3Bucket, c.StepsDirPath); err != nil {
		failf("Error: ", err)
	}

	fmt.Println()
	fmt.Println("==> Uploading spec.json")
	if err := uploadFileToS3(c.S3Bucket, c.CollectionSpecJSONPath, "/spec.json"); err != nil {
		failf("Error: ", err)
	}
	fmt.Println()
	fmt.Println("==> Uploading slim-spec.json")
	if err := uploadFileToS3(c.S3Bucket, c.CollectionSlimSpecJSONPath, "/slim-spec.json", "--content-encoding", "gzip"); err != nil {
		failf("Error: ", err)
	}

	fmt.Println("==> DONE")
}
