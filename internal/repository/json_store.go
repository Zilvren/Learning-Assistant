package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// dataDir 是放数据文件的目录，相对于运行程序的目录
var dataDir = "data"

// SetDataDir 设置数据目录（可选的，默认就是 "data"）
func SetDataDir(dir string) {
	dataDir = dir
}

// ensureDataDir 确保 data 目录存在
func ensureDataDir() error {
	return os.MkdirAll(dataDir, 0755)
}

// LoadJSON 从 data 目录读取 JSON 文件，解析到 target 里
// target 必须是一个指针，比如 &subjects 或 &errors
// 如果文件不存在，把 target 置为空（空切片/空结构体），不会报错
func LoadJSON(filename string, target interface{}) error {
	path := filepath.Join(dataDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在不算错
		}
		return err
	}
	return json.Unmarshal(data, target)
}

// SaveJSON 把数据写入 data 目录的 JSON 文件
// indent 让文件可读性好（格式化输出）
func SaveJSON(filename string, data interface{}) error {
	if err := ensureDataDir(); err != nil {
		return err
	}
	path := filepath.Join(dataDir, filename)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// DataDir 返回数据目录路径，如果目录不存在则自动创建
func DataDir() string {
	_ = os.MkdirAll(dataDir, 0755)
	return dataDir
}

// Path 返回数据文件的完整路径，如果目录不存在则自动创建
func Path(filename string) string {
	_ = os.MkdirAll(dataDir, 0755)
	return filepath.Join(dataDir, filename)
}
