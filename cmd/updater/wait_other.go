//go:build !windows

package main

import (
	"os"
	"syscall"
	"time"
)

// waitForPID 在命令行工具中完成本文件定义的局部处理。
func waitForPID(pid int, timeout time.Duration) {
	if pid <= 0 {
		return
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}
