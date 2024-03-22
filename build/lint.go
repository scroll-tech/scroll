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

func main() {
	log.SetFlags(log.Lshortfile)

	if _, err := os.Stat(filepath.Join("..", "build", "lint.go")); os.IsNotExist(err) {
		log.Fatal("should run build from root dir")
	}

	lint()
}

func lint() {
	v := flag.Bool("v", false, "log verbosely")
	flag.Parse()

	// Make sure GOLANGCI is downloaded and available.
	if err := goGet(GolangCIVersion); err != nil {
		log.Fatalf("could not get golangci-lint: %v", err)
	}

	cmd := exec.Command(filepath.Join(goBin(), "golangci-lint"))
	cmd.Args = append(cmd.Args, "run", "--config", "../build/.golangci.yml")

	if *v {
		cmd.Args = append(cmd.Args, "-v")
	}

	fmt.Println("Linting...")
	cmd.Stderr, cmd.Stdout = os.Stderr, os.Stdout

	if err := cmd.Run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func goBin() string {
	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		log.Fatal("GOBIN not set")
	}
	return gobin
}

func goGet(pkg string) error {
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "install", pkg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not install %s: %v\n%s", pkg, err, string(out))
	}
	return nil
}
