package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/shelley"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload the latest binary to the container registry",
	Run:   runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(cmd *cobra.Command, args []string) {
	outputPath, err := rootState.BinaryPath(rootConfig.Project.Name)
	if err != nil {
		log.Fatal(err)
	}

	stat, err := os.Stat(outputPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		log.Fatal("must build a binary before uploading")
	case err != nil:
		log.Fatal(err)
	case !stat.Mode().IsRegular():
		log.Fatalf("%s is not a regular file", outputPath)
	}

	outputHash, err := fileSHA256(outputPath)
	if err != nil {
		log.Fatal(err)
	}

	repository, _ := shelley.
		Command(
			"aws", "ecr", "describe-repositories",
			"--repository-names", rootConfig.Repository.Name,
			"--query", "repositories[0].repositoryUri", "--output", "text",
		).
		Debug().
		ErrExit().
		Text()

	_ = strings.SplitN(repository, "/", 2)[0]
	image := repository + ":" + outputHash

	authenticated, _ := shelley.
		Command("go", "run", "go.alexhamlin.co/zeroimage@main", "check-auth", "--push", image).
		Debug().
		ErrExit().
		NoOutput().
		Successful()

	if !authenticated {
		log.Fatal("[need to authenticate]")
	}

	log.Print("[ready to push]")
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
