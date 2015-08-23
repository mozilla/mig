// This package contains utility methos for testing
package test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func GetTestCasePath() string {
	//TODO: Right now the command is hardcoded. We should decide how to fix this.
	dirPath, err := filepath.Abs("../test/tools")
	testFileName := "test"

	// Ugly hack for windows: The testFileName must end in .exe
	if runtime.GOOS == "windows" {
		testFileName = "test.exe"
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}

	path, err := filepath.EvalSymlinks(filepath.Join(dirPath, testFileName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}

	return path
}

func PrintSoftErrors(softerrors []error) {
	for _, err := range softerrors {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

// this method redirects the process's stdout to the test stdout
func LaunchTestCase() (*exec.Cmd, error) {
	cmd := exec.Command(GetTestCasePath())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	return cmd, err
}

func LaunchTestCaseAndWaitForInitialization() (*exec.Cmd, error) {
	return launchProcessAndWaitInitialization(GetTestCasePath())
}

// starts a process and waits until it writes everythin to stdout: that way we know it has been initialized.
// the process launched should close stdout once it has been fully initialized.
// this method redirects the process's stdout to the test stdout
func launchProcessAndWaitInitialization(file string) (*exec.Cmd, error) {
	cmd := exec.Command(file)

	childout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer childout.Close()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	io.Copy(os.Stdout, childout)

	return cmd, nil
}
