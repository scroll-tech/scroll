//go:build none
// +build none

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	// GolangCIVersion to be used for linting.
	GolangCIVersion = "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2"
)

// GOBIN environment variable.
func goBin() string {
	if os.Getenv("GOBIN") == "" {
		log.Fatal("GOBIN not set")
	}

	return os.Getenv("GOBIN")
}

func main() {
	log.SetFlags(log.Lshortfile)

	if _, err := os.Stat(filepath.Join("../build", "lint.go")); os.IsNotExist(err) {
		log.Fatal("should run build from root dir")
	}

	lint()
}

//nolint:gosec
func lint() {
	v := flag.Bool("v", false, "log verbosely")

	// Make sure GOLANGCI is downloaded and available.
	argsGet := []string{"install", GolangCIVersion}
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), argsGet...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("could not list pkgs: %v\n%s", err, string(out))
	}

	cmd = exec.Command(filepath.Join(goBin(), "golangci-lint"))
	cmd.Args = append(cmd.Args, "run", "--config", "../build/.golangci.yml")

	if *v {
		cmd.Args = append(cmd.Args, "-v")
	}

	fmt.Println("Linting...")
	cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout

	if err := cmd.Run(); err != nil {
		log.Fatal("Error: Could not Lint ", "error: ", err, ", cmd: ", cmd)
	}
}
