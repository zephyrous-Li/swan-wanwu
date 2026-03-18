// Package persistent 提供会话持久化的布局管理。
package persistent

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/UnicomAI/wanwu/pkg/wga-persistent/internal/storage"
)

// ============================================================================
// Mode
// ============================================================================

// Mode 持久化存储模式。
type Mode string

const (
	// ModeOverwrite 覆盖模式：每次执行覆盖同一目录。
	ModeOverwrite Mode = "overwrite"
	// ModeVersioned 分轮存储模式：每次执行创建独立的 run 目录。
	ModeVersioned Mode = "versioned"
)

// ============================================================================
// SessionDirInfo
// ============================================================================

// SessionDirInfo 会话目录信息。
type SessionDirInfo struct {
	ThreadID  string // 会话 ID
	RunID     string // 执行 ID（overwrite 模式为空）
	Mode      Mode   // 存储模式
	Dir       string // 完整路径
	Timestamp int64  // 毫秒时间戳（overwrite 模式为 0，versioned 模式为目录创建时间）
}

// ============================================================================
// Option
// ============================================================================

// Option 目录操作选项。
type Option func(*dirConfig)

type dirConfig struct {
	mkdir          bool
	copyLastOutput bool
	perm           os.FileMode
}

// WithMkdir 创建目录（如果不存在）。
// copyLastOutput: 是否从上一次输出复制（仅 versioned 模式有效）
// perm: 目录权限，默认 0755
func WithMkdir(copyLastOutput bool, perm ...os.FileMode) Option {
	p := os.FileMode(0755)
	if len(perm) > 0 {
		p = perm[0]
	}
	return func(c *dirConfig) {
		c.mkdir = true
		c.copyLastOutput = copyLastOutput
		c.perm = p
	}
}

func applyOptions(opts ...Option) *dirConfig {
	c := &dirConfig{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ============================================================================
// SessionPersistent Interface
// ============================================================================

// SessionPersistent 会话持久化接口。
type SessionPersistent interface {
	// GetThreadDir 获取 session 目录信息。
	GetThreadDir() SessionDirInfo
	// GetRunDir 获取指定 run 的保存目录。
	// opts: WithMkdir 创建目录（如果不存在）
	// ok=true: 目录存在
	// ok=false: 目录不存在
	GetRunDir(runID string, opts ...Option) (ok bool, info SessionDirInfo, err error)
	// GetLastRunDir 获取最新的恢复目录。
	GetLastRunDir() (ok bool, info SessionDirInfo, err error)
	// ListRunDirs 列出所有 run 目录。
	ListRunDirs() ([]SessionDirInfo, error)
	// CleanupRun 清理指定 run。
	CleanupRun(runID string) error
	// Cleanup 清理整个 session。
	Cleanup() error
}

// ============================================================================
// Errors
// ============================================================================

var (
	// ErrModeConflict 目录已存在但 mode 不匹配。
	ErrModeConflict = errors.New("persistent mode conflict with existing directory")
)

// ============================================================================
// NewSessionPersistent
// ============================================================================

// NewSessionPersistent 创建会话持久化实例。
// 如果目录已存在但 mode 与传入的 mode 不同，返回 ErrModeConflict。
func NewSessionPersistent(mode Mode, baseDir string, threadID string) (SessionPersistent, error) {
	s := storage.NewLocalStorage(baseDir)

	detectedMode, err := detectMode(s, threadID)
	if err != nil {
		return nil, err
	}
	if detectedMode != "" && detectedMode != mode {
		return nil, fmt.Errorf("%w: existing=%s, requested=%s", ErrModeConflict, detectedMode, mode)
	}

	switch mode {
	case ModeVersioned:
		return newVersionedPersistent(s, threadID), nil
	default:
		return newOverwritePersistent(s, threadID), nil
	}
}

// ============================================================================
// 辅助函数 - 公共
// ============================================================================

const (
	threadDirPrefix = "thread-"
	dirSeparator    = "_"
)

// buildThreadDir 构建 thread 目录名：thread-{mode}_{threadID}
func buildThreadDir(threadID string, mode Mode) string {
	return threadDirPrefix + string(mode) + dirSeparator + threadID
}

// detectMode 检测已存在目录的 mode。
// 使用 ListDirs + 前缀匹配，只需一次 I/O。
func detectMode(s storage.Storage, threadID string) (Mode, error) {
	dirs, err := s.ListDirs(s.BaseDir())
	if err != nil {
		return "", err
	}

	overwritePrefix := threadDirPrefix + string(ModeOverwrite) + dirSeparator + threadID
	versionedPrefix := threadDirPrefix + string(ModeVersioned) + dirSeparator + threadID

	for _, dir := range dirs {
		if strings.HasPrefix(dir, overwritePrefix) {
			return ModeOverwrite, nil
		}
		if strings.HasPrefix(dir, versionedPrefix) {
			return ModeVersioned, nil
		}
	}

	return "", nil
}

// ============================================================================
// 复制函数
// ============================================================================

// copyDir 复制目录内容到目标目录。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile 复制文件。
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
