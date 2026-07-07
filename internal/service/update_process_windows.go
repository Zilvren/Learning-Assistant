//go:build windows

package service

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
)

func startUpdaterProcess(updaterPath, packagePath, appDir, appExe string, pid int) error {
	args := updateProcessArgs(packagePath, appDir, appExe, pid)
	cmd := exec.Command(updaterPath, args...)
	cmd.Dir = appDir
	if err := cmd.Start(); err != nil {
		if isElevationRequired(err) {
			return shellExecuteRunas(updaterPath, strings.Join(quoteWindowsArgs(args), " "), appDir)
		}
		return err
	}
	return cmd.Process.Release()
}

func updateProcessArgs(packagePath, appDir, appExe string, pid int) []string {
	return []string{
		"--package", packagePath,
		"--app-dir", appDir,
		"--app-exe", appExe,
		"--pid", strconv.Itoa(pid),
	}
}

func isElevationRequired(err error) bool {
	var exitErr *exec.Error
	if errors.As(err, &exitErr) {
		err = exitErr.Err
	}
	return errors.Is(err, windows.ERROR_ELEVATION_REQUIRED)
}

func shellExecuteRunas(file, args, cwd string) error {
	verbPtr, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	filePtr, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	argsPtr, err := windows.UTF16PtrFromString(args)
	if err != nil {
		return err
	}
	cwdPtr, err := windows.UTF16PtrFromString(cwd)
	if err != nil {
		return err
	}
	if err := windows.ShellExecute(0, verbPtr, filePtr, argsPtr, cwdPtr, windows.SW_HIDE); err != nil {
		return fmt.Errorf("需要管理员权限启动更新器，但提权启动失败：%w", err)
	}
	return nil
}

func quoteWindowsArgs(args []string) []string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, quoteWindowsArg(arg))
	}
	return quoted
}

func quoteWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\n\v\"") {
		return arg
	}

	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range arg {
		if r == '\\' {
			backslashes++
			continue
		}
		if r == '"' {
			b.WriteString(strings.Repeat(`\`, backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
			continue
		}
		if backslashes > 0 {
			b.WriteString(strings.Repeat(`\`, backslashes))
			backslashes = 0
		}
		b.WriteRune(r)
	}
	if backslashes > 0 {
		b.WriteString(strings.Repeat(`\`, backslashes*2))
	}
	b.WriteByte('"')
	return b.String()
}
