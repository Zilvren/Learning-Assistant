package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const defaultJSONLockTimeout = 10 * time.Second

var (
	ErrDataBusy = errors.New("数据目录正在被其他操作占用")
	errLockBusy = errors.New("file lock busy")

	dataDirMu sync.RWMutex
	dataDir   = "data"

	storeRegistryMu sync.Mutex
	storeRegistry   = map[string]*sync.RWMutex{}
)

// JSONStore serializes access to one data directory, both within this process
// and across different Tracker processes.
type JSONStore struct {
	dir         string
	local       *sync.RWMutex
	lockTimeout time.Duration
}

// JSONTx is valid only for the duration of a JSONStore Read or Write callback.
type JSONTx struct {
	dir      string
	writable bool
}

// SetDataDir 在存储层中完成本文件定义的局部处理。
func SetDataDir(dir string) {
	dataDirMu.Lock()
	dataDir = normalizeDataDir(dir)
	dataDirMu.Unlock()
}

// DataDir 在存储层中完成本文件定义的局部处理。
func DataDir() string {
	dataDirMu.RLock()
	dir := normalizeDataDir(dataDir)
	dataDirMu.RUnlock()
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// Path 在存储层中完成本文件定义的局部处理。
func Path(filename string) string {
	return filepath.Join(DataDir(), filepath.Base(filename))
}

// StoreBlob 在存储层中完成本文件定义的局部处理。
func StoreBlob(r io.Reader) (string, int64, error) {
	tmpDir := filepath.Join(DataDir(), "blobs", ".tmp")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(tmpDir, "blob-*")
	if err != nil {
		return "", 0, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, h), r)
	if err != nil {
		tmp.Close()
		return "", 0, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, err
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))
	target := BlobPath(hash)
	if _, err := os.Stat(target); err == nil {
		return hash, size, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return "", 0, err
	}
	if err := replaceFileAtomic(tmpPath, target); err != nil {
		if _, statErr := os.Stat(target); statErr == nil {
			return hash, size, nil
		}
		return "", 0, err
	}
	return hash, size, nil
}

// BlobPath 在存储层中完成本文件定义的局部处理。
func BlobPath(hash string) string {
	if len(hash) < 2 {
		return filepath.Join(DataDir(), "blobs", "invalid")
	}
	return filepath.Join(DataDir(), "blobs", hash[:2], hash)
}

// ReadBlob 在存储层中读取并整理所需数据。
func ReadBlob(hash string) ([]byte, error) { return os.ReadFile(BlobPath(hash)) }

// NewJSONStore 在存储层中创建所需对象并完成初始化。
func NewJSONStore(dir string) *JSONStore {
	dir = normalizeDataDir(dir)
	storeRegistryMu.Lock()
	local := storeRegistry[dir]
	if local == nil {
		local = &sync.RWMutex{}
		storeRegistry[dir] = local
	}
	storeRegistryMu.Unlock()
	return &JSONStore{
		dir:         dir,
		local:       local,
		lockTimeout: defaultJSONLockTimeout,
	}
}

// DefaultJSONStore 在存储层中完成本文件定义的局部处理。
func DefaultJSONStore() *JSONStore {
	return NewJSONStore(DataDir())
}

// Read 在存储层中读取并整理所需数据。
func (s *JSONStore) Read(ctx context.Context, fn func(*JSONTx) error) error {
	return s.withLock(ctx, false, fn)
}

// Write 在存储层中创建或更新相应状态。
func (s *JSONStore) Write(ctx context.Context, fn func(*JSONTx) error) error {
	return s.withLock(ctx, true, fn)
}

// withLock 在存储层中完成本文件定义的局部处理。
func (s *JSONStore) withLock(ctx context.Context, write bool, fn func(*JSONTx) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := s.lockTimeout
	if timeout <= 0 {
		timeout = defaultJSONLockTimeout
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := s.acquireLocal(waitCtx, write); err != nil {
		return err
	}
	if write {
		defer s.local.Unlock()
	} else {
		defer s.local.RUnlock()
	}

	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	release, err := acquireFileLock(waitCtx, filepath.Join(s.dir, ".tracker-data.lock"), write)
	if err != nil {
		return err
	}
	defer release()

	return fn(&JSONTx{dir: s.dir, writable: write})
}

// acquireLocal 在存储层中完成本文件定义的局部处理。
func (s *JSONStore) acquireLocal(ctx context.Context, write bool) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var acquired bool
		if write {
			acquired = s.local.TryLock()
		} else {
			acquired = s.local.TryRLock()
		}
		if acquired {
			return nil
		}
		select {
		case <-ctx.Done():
			return dataLockError(ctx.Err())
		case <-ticker.C:
		}
	}
}

// acquireFileLock 在存储层中完成本文件定义的局部处理。
func acquireFileLock(ctx context.Context, path string, exclusive bool) (func(), error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		release, err := tryFileLock(path, exclusive)
		if err == nil {
			return release, nil
		}
		if !errors.Is(err, errLockBusy) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, dataLockError(ctx.Err())
		case <-ticker.C:
		}
	}
}

// dataLockError 在存储层中完成本文件定义的局部处理。
func dataLockError(cause error) error {
	return fmt.Errorf("%w: %v", ErrDataBusy, cause)
}

// Load 在存储层中读取并整理所需数据。
func (tx *JSONTx) Load(filename string, target interface{}) error {
	path, err := tx.path(filename)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, target)
}

// Save 在存储层中创建或更新相应状态。
func (tx *JSONTx) Save(filename string, value interface{}) error {
	if !tx.writable {
		return errors.New("JSON read transaction cannot write")
	}
	path, err := tx.path(filename)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0600)
}

// SaveAll marshals all values before replacing any file, then rolls back files
// already replaced if a later filesystem write fails. It gives JSON backup
// imports all-or-nothing behavior for normal I/O failures.
func (tx *JSONTx) SaveAll(values map[string]interface{}) error {
	if !tx.writable {
		return errors.New("JSON read transaction cannot write")
	}
	type preparedFile struct {
		name   string
		path   string
		data   []byte
		exists bool
		before []byte
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	prepared := make([]preparedFile, 0, len(names))
	for _, name := range names {
		path, err := tx.path(name)
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(values[name], "", "  ")
		if err != nil {
			return err
		}
		before, readErr := os.ReadFile(path)
		if readErr != nil && !os.IsNotExist(readErr) {
			return readErr
		}
		prepared = append(prepared, preparedFile{name: name, path: path, data: data, exists: readErr == nil, before: before})
	}
	committed := make([]preparedFile, 0, len(prepared))
	for _, item := range prepared {
		if err := writeFileAtomic(item.path, item.data, 0600); err != nil {
			for index := len(committed) - 1; index >= 0; index-- {
				prior := committed[index]
				if prior.exists {
					_ = writeFileAtomic(prior.path, prior.before, 0600)
				} else {
					_ = os.Remove(prior.path)
				}
			}
			return err
		}
		committed = append(committed, item)
	}
	return nil
}

// path 在存储层中完成本文件定义的局部处理。
func (tx *JSONTx) path(filename string) (string, error) {
	if filename == "" || filepath.Base(filename) != filename {
		return "", fmt.Errorf("无效 JSON 文件名：%s", filename)
	}
	return filepath.Join(tx.dir, filename), nil
}

// writeFileAtomic 在存储层中创建或更新相应状态。
func writeFileAtomic(target string, data []byte, mode os.FileMode) (err error) {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return replaceFileAtomic(tmpPath, target)
}

// normalizeDataDir 在存储层中构造、编码或标准化数据。
func normalizeDataDir(dir string) string {
	if dir == "" {
		dir = "data"
	}
	absolute, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		return filepath.Clean(dir)
	}
	return absolute
}

// LoadJSON and SaveJSON remain for compatibility. Read-modify-write callers
// should use one JSONStore transaction instead of calling these separately.
// LoadJSON 以兼容方式读取一个 JSON 数据文件到目标对象。
func LoadJSON(filename string, target interface{}) error {
	return DefaultJSONStore().Read(context.Background(), func(tx *JSONTx) error {
		return tx.Load(filename, target)
	})
}

// SaveJSON 在存储层中创建或更新相应状态。
func SaveJSON(filename string, value interface{}) error {
	return DefaultJSONStore().Write(context.Background(), func(tx *JSONTx) error {
		return tx.Save(filename, value)
	})
}
