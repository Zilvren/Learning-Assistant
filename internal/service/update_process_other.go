//go:build !windows

package service

import (
	"os/exec"
	"strconv"
)

func startUpdaterProcess(updaterPath, packagePath, appDir, appExe string, pid int) error {
	cmd := exec.Command(
		updaterPath,
		"--package", packagePath,
		"--app-dir", appDir,
		"--app-exe", appExe,
		"--pid", strconv.Itoa(pid),
	)
	cmd.Dir = appDir
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
