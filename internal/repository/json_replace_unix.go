//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package repository

import (
	"os"
	"path/filepath"
)

// replaceFileAtomic 在存储层中创建或更新相应状态。
func replaceFileAtomic(source string, target string) error {
	if err := os.Rename(source, target); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(target))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
