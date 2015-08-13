package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var (
	awsAccessKey           = ""
	awsSecretKey           = ""
	s3Bucket               = ""
	s3BucketRegion         = ""
	stepsDirPath           = ""
	collectionRootDirPath  = ""
	collectionSpecJSONPath = ""
	//
	s3cmdConfigFilePath          = ""
	stepsAssetsBasePathInBucket  = "steps"
	stepArchivesBasePathInBucket = "step-archives"
)

// ------------------------------------------------
// --- UTILS

func writeStringToFile(pth string, fileCont string) error {
	return writeBytesToFile(pth, []byte(fileCont))
}

func writeBytesToFile(pth string, fileCont []byte) error {
	if pth == "" {
		return errors.New("No path provided")
	}

	file, err := os.Create(pth)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(" (!) Failed to close file:", err)
		}
	}()

	if _, err := file.Write(fileCont); err != nil {
		return err
	}

	return nil
}

func runCommandReturnCombinedOut(name string, args ...string) (out []byte, err error) {
	cmd := exec.Command(name, args...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf(" [!] Command failed: %#v\n", cmd)
		return
	}
	return
}
func runCommandInDir(cmdDir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if cmdDir != "" {
		cmd.Dir = cmdDir
	}

	if err := cmd.Run(); err != nil {
		log.Printf(" [!] Command failed: %#v\n", cmd)
		return err
	}
	return nil
}
func runCommand(name string, args ...string) error {
	return runCommandInDir("", name, args...)
}

func prepareTmpStepZIPArchiveFromGit(stepID, stepVersion string) (string, error) {
	stepVerTmpDir, err := ioutil.TempDir("", stepID+"."+stepVersion+"--src")
	if err != nil {
		return "", err
	}
	stepZipArchivePth, err := ioutil.TempDir("", stepID+"."+stepVersion+"--zip")
	if err != nil {
		return "", err
	}
	stepZipArchivePth += "/step.zip"
	log.Println("stepVerTmpDir: ", stepVerTmpDir)
	log.Println("stepZipArchivePth: ", stepZipArchivePth)

	// stepman activate into the tmp dir
	if err := runCommand("stepman", "activate", "--collection", collectionRootDirPath, "--id", stepID, "--version", stepVersion, "--path", stepVerTmpDir); err != nil {
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
	if err := runCommandInDir(stepVerTmpDir, "zip", "-r", stepZipArchivePth, "."); err != nil {
		return "", err
	}

	return stepZipArchivePth, nil
}

func uploadFileToS3(filePth, pathInBucket string) error {
	upldPth := "s3://" + s3Bucket
	if !strings.HasPrefix(pathInBucket, "/") {
		upldPth += "/"
	}
	upldPth += pathInBucket

	log.Println("Uploading file: ", filePth, " | to: ", upldPth)
	if output, err := runCommandReturnCombinedOut("s3cmd", "--debug", "-c", s3cmdConfigFilePath, "--acl-public", "put", filePth, upldPth); err != nil {
		fmt.Println()
		fmt.Println()
		log.Println("==========> [!] FAILED to upload file")
		log.Println("=> OUTPUT:")
		log.Printf("%s\n", output)
		fmt.Println()
		fmt.Println()
		return err
	}
	return nil
}

// ------------------------------------------------
// --- MAIN

func createS3cmdConfigFileContent(accessKey, secretKey, bucketRegion string) string {
	formatStr := `[default]
access_key = %s
secret_key = %s
`
	confCont := fmt.Sprintf(formatStr, accessKey, secretKey)
	if bucketRegion != "" {
		confCont += "\nbucket_location = " + bucketRegion
	}
	return confCont
}

func uploadStepsDir() error {
	s3UploadFullPath := "s3://" + s3Bucket + "/" + stepsAssetsBasePathInBucket + "/"
	upStepsDirPth := stepsDirPath
	if !strings.HasSuffix(upStepsDirPth, "/") {
		upStepsDirPth += "/"
	}
	log.Println("Uploading from: ", upStepsDirPth, " | to: ", s3UploadFullPath)
	if err := runCommand("s3cmd", "-c", s3cmdConfigFilePath, "--acl-public", "--recursive", "sync", upStepsDirPth, s3UploadFullPath); err != nil {
		return err
	}
	return nil
}

func uploadStepArchiveZIPs() error {
	// s3UploadBasePath := "s3://" + s3Bucket + "/" + stepArchivesBasePathInBucket
	s3ZipCheckBasePath := "https://" + s3Bucket + ".s3.amazonaws.com/" + stepArchivesBasePathInBucket

	stepYmlPattern := stepsDirPath + "/*/*/step.yml"
	log.Println("stepYmlPattern: ", stepYmlPattern)
	stepYmlPaths, err := filepath.Glob(stepYmlPattern)
	if err != nil {
		return err
	}
	log.Println("Found step count: ", len(stepYmlPaths))
	for idx, aStepYmlPth := range stepYmlPaths {
		fmt.Println()
		log.Printf("---> %d: %s", idx, aStepYmlPth)
		pathComps := strings.Split(aStepYmlPth, "/")
		if len(pathComps) < 4 {
			log.Println(" [!] The step.yml path should have at least 4 path components (steps, step-id, step-version, step.yml): ", aStepYmlPth)
			return fmt.Errorf("Invalid step.yml path: %s", aStepYmlPth)
		}
		pathComps = pathComps[len(pathComps)-4:]
		if pathComps[0] != "steps" {
			log.Println(" [!] Invalid step.yml path, 'steps' should be included right before the step-id: ", aStepYmlPth)
			return fmt.Errorf("Invalid step.yml path: %s", aStepYmlPth)
		}
		if pathComps[3] != "step.yml" {
			log.Println(" [!] Invalid step.yml path, should end with 'step.yml': ", aStepYmlPth)
			return fmt.Errorf("Invalid step.yml path: %s", aStepYmlPth)
		}
		stepID := pathComps[1]
		stepVersion := pathComps[2]

		// check if ZIP for this step&version already exists
		stepZipCheckURL := s3ZipCheckBasePath + "/" + stepID + "/" + stepVersion + "/step.zip"
		log.Println("Checking: ", stepZipCheckURL)
		resp, err := http.Head(stepZipCheckURL)
		if err != nil {
			return err
		}
		if resp.StatusCode == 200 {
			log.Println("---> zip found [OK]")
			continue
		}

		log.Println("zip NOT found or not accessible - creating and uploading...")

		zipPth, err := prepareTmpStepZIPArchiveFromGit(stepID, stepVersion)
		if err != nil {
			return err
		}
		log.Println("zipPth: ", zipPth)
		zipPthInBucket := stepArchivesBasePathInBucket + "/" + stepID + "/" + stepVersion + "/step.zip"
		if err := uploadFileToS3(zipPth, zipPthInBucket); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	flag.StringVar(&awsAccessKey, "accesskey", "", "AWS Access Key")
	flag.StringVar(&awsSecretKey, "secretkey", "", "AWS Secret Key")
	flag.StringVar(&s3Bucket, "s3bucket", "", "AWS S3 Bucket name")
	flag.StringVar(&stepsDirPath, "stepsdir", "", "steps directory path")
	flag.StringVar(&collectionRootDirPath, "collection-root-dir", "", "Collection's root dir path")
	flag.StringVar(&collectionSpecJSONPath, "collection-spec-json", "", "Collection's spec.json file's path")
	// optional
	flag.StringVar(&s3BucketRegion, "s3bucket-region", "", "AWS S3 Bucket region")
}

func main() {
	// --- INPUTS
	flag.Parse()
	if awsAccessKey == "" {
		log.Fatalln("AWS Access Key not provided!")
	}
	if awsSecretKey == "" {
		log.Fatalln("AWS Secret Key not provided!")
	}
	if s3Bucket == "" {
		log.Fatalln("AWS S3 Bucket name not provided!")
	}
	if stepsDirPath == "" {
		log.Fatalln("steps directory path not provided!")
	}
	if collectionRootDirPath == "" {
		log.Fatalln("Collection's root dir path not provided!")
	}
	if collectionSpecJSONPath == "" {
		log.Fatalln("Collection's spec.json file's path not provided!")
	}

	currWorkDir, err := filepath.Abs("./")
	if err != nil {
		log.Fatalln("Failed to get current working dir path: ", err)
	}
	s3cmdConfigFilePath = path.Join(currWorkDir, "s3cmd.conf")

	// --- START of main logic
	if err := writeStringToFile(s3cmdConfigFilePath, createS3cmdConfigFileContent(awsAccessKey, awsSecretKey, s3BucketRegion)); err != nil {
		log.Fatalln("Error: ", err)
	}

	fmt.Println()
	log.Println("==> Uploading steps/ dir...")
	fmt.Println()
	if err := uploadStepsDir(); err != nil {
		log.Fatalln("Error: ", err)
	}

	fmt.Println()
	log.Println("==> Uploading step ZIPs...")
	fmt.Println()
	if err := uploadStepArchiveZIPs(); err != nil {
		log.Fatalln("Error: ", err)
	}

	// upload the spec.json as well
	fmt.Println()
	log.Println("==> Uploading spec.json")
	if err := uploadFileToS3(collectionSpecJSONPath, "/spec.json"); err != nil {
		log.Fatalln("Error: ", err)
	}

	fmt.Println("==> DONE")
}
