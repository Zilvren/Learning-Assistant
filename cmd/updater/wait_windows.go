//go:build windows

package main

import (
	"time"

	"golang.org/x/sys/windows"
)

// waitForPID 在命令行工具中完成本文件定义的局部处理。
func waitForPID(pid int, timeout time.Duration) {
	if pid <= 0 {
		return
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)
	_, _ = windows.WaitForSingleObject(handle, uint32(timeout/time.Millisecond))
}
