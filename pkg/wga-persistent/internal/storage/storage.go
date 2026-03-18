// Package storage 提供存储后端接口。
package storage

import "os"

// Storage 存储后端接口。
type Storage interface {
	// Exists 判断路径是否存在。
	Exists(path string) (bool, error)
	// ListDirs 列出目录下的子目录名称。
	ListDirs(path string) ([]string, error)
	// Mkdir 创建目录（包括父目录）。
	Mkdir(path string, perm os.FileMode) error
	// Remove 删除路径。
	Remove(path string) error
	// BaseDir 返回基础目录。
	BaseDir() string
}

// localStorage 本地文件系统存储实现。
type localStorage struct {
	baseDir string
}

// NewLocalStorage 创建本地文件系统存储。
func NewLocalStorage(baseDir string) Storage {
	return &localStorage{baseDir: baseDir}
}

func (s *localStorage) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *localStorage) ListDirs(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs, nil
}

func (s *localStorage) Remove(path string) error {
	return os.RemoveAll(path)
}

func (s *localStorage) Mkdir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (s *localStorage) BaseDir() string {
	return s.baseDir
}
