package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionUsesReleaseInjectionWithoutStartingApplication(testingContext *testing.T) {
	directory := testingContext.TempDir()
	binary := filepath.Join(directory, "ssh-keyflow")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-buildvcs=false", "-ldflags", "-X main.version=v1.2.3-test", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		testingContext.Fatalf("build: %v %s", err, output)
	}
	command := exec.Command(binary, "--version")
	for _, entry := range os.Environ() {
		upper := strings.ToUpper(entry)
		if !strings.HasPrefix(upper, "HOME=") && !strings.HasPrefix(upper, "USERPROFILE=") {
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, "HOME="+directory, "USERPROFILE="+directory)
	output, err := command.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "SSH KeyFlow v1.2.3-test" {
		testingContext.Fatalf("version: %v %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(directory, ".ssh")); !os.IsNotExist(err) {
		testingContext.Fatal("--version initialized SSH storage")
	}
}
