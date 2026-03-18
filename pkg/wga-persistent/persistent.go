// Package wga_persistent 提供会话持久化存储管理能力。
//
// 该包支持两种存储模式：
//   - ModeOverwrite: 覆盖模式，每次执行覆盖同一目录
//   - ModeVersioned: 分轮存储模式，每次执行创建独立的 run 目录
//
// # 目录命名规则
//
// thread 目录: thread-{mode}_{threadID}
//   - 例如: thread-overwrite_abc123, thread-versioned_xyz789
//
// run 目录 (仅 versioned 模式): run-{timestamp}_{runID}
//   - 例如: run-1709523600000_run001
//
// # 并发安全
//
// Store 是并发安全的。多个 goroutine 可以同时调用 Store 的方法，
// 内部使用读写锁保护共享状态。
//
// 使用 WithMkdir 选项可以确保 GetRunDir 在并发场景下对同一 runID
// 返回相同的目录：
//
//	ok, info, err := store.GetRunDir("run-1", wga_persistent.WithMkdir(false))
//	// ok=true: 目录存在
//
// 使用 WithMkdir(true) 可以在创建目录的同时从上一次输出复制：
//
//	ok, info, err := store.GetRunDir("run-1", wga_persistent.WithMkdir(true))
//	// 创建目录并复制上一次的输出
package wga_persistent

import (
	"os"

	"github.com/UnicomAI/wanwu/pkg/wga-persistent/internal/persistent"
)

// ============================================================================
// Mode
// ============================================================================

// Mode 持久化存储模式。
type Mode = persistent.Mode

const (
	// ModeOverwrite 覆盖模式：每次执行覆盖同一目录。
	ModeOverwrite = persistent.ModeOverwrite
	// ModeVersioned 分轮存储模式：每次执行创建独立的 run 目录。
	ModeVersioned = persistent.ModeVersioned
)

// ============================================================================
// Errors
// ============================================================================

var (
	// ErrModeConflict 目录已存在但 mode 不匹配。
	ErrModeConflict = persistent.ErrModeConflict
)

// ============================================================================
// Option
// ============================================================================

// Option 目录操作选项。
type Option = persistent.Option

// WithMkdir 创建目录（如果不存在）。
// copyLastOutput: 是否从上一次输出复制（仅 versioned 模式有效）
// perm: 目录权限，默认 0755
func WithMkdir(copyLastOutput bool, perm ...os.FileMode) Option {
	return persistent.WithMkdir(copyLastOutput, perm...)
}

// ============================================================================
// SessionDirInfo
// ============================================================================

// SessionDirInfo 会话目录信息。
type SessionDirInfo = persistent.SessionDirInfo

// ============================================================================
// Store
// ============================================================================

// Store 持久化存储管理器，绑定一个 session。
//
// Store 是并发安全的，可以在多个 goroutine 中同时使用。
type Store struct {
	impl persistent.SessionPersistent
}

// NewStore 创建持久化存储管理器。
// 如果目录已存在但 mode 与传入的 mode 不同，返回 ErrModeConflict。
//
// threadID 可以包含任意字符，用于标识一个会话线程。
func NewStore(mode Mode, baseDir string, threadID string) (*Store, error) {
	impl, err := persistent.NewSessionPersistent(mode, baseDir, threadID)
	if err != nil {
		return nil, err
	}
	return &Store{impl: impl}, nil
}

// GetThreadDir 获取 session 目录信息。
func (s *Store) GetThreadDir() SessionDirInfo {
	return s.impl.GetThreadDir()
}

// GetRunDir 获取指定 run 的保存目录。
//
// 对于 overwrite 模式，runID 参数被忽略，返回 thread 目录。
// 对于 versioned 模式，如果 run 目录已存在则返回已存在的目录信息，
// 否则生成新的目录名（包含当前时间戳）。
//
// 选项：
//   - WithMkdir(false): 创建目录（如果不存在），权限默认 0755
//   - WithMkdir(false, 0755): 创建目录，指定权限
//   - WithMkdir(true): 创建目录并从上一次输出复制，权限默认 0755
//   - WithMkdir(true, 0755): 创建目录并复制，指定权限
//
// 返回值：
//   - ok: 目录是否存在（使用 WithMkdir 创建后返回 true）
//   - info: 目录信息（无论是否存在都有意义）
//   - err: 错误信息
func (s *Store) GetRunDir(runID string, opts ...Option) (ok bool, info SessionDirInfo, err error) {
	return s.impl.GetRunDir(runID, opts...)
}

// GetLastRunDir 获取最新的恢复目录。
//
// 对于 overwrite 模式，返回 thread 目录。
// 对于 versioned 模式，返回时间戳最新的 run 目录。
//
// 返回值：
//   - ok: 是否存在可恢复的目录
//   - info: 目录信息（ok=false 时为零值）
//   - err: 错误信息
func (s *Store) GetLastRunDir() (ok bool, info SessionDirInfo, err error) {
	return s.impl.GetLastRunDir()
}

// ListRunDirs 列出所有 run 目录。
//
// 对于 overwrite 模式，如果目录存在则返回单元素列表，否则返回空列表。
// 对于 versioned 模式，按时间戳降序返回所有 run 目录。
func (s *Store) ListRunDirs() ([]SessionDirInfo, error) {
	return s.impl.ListRunDirs()
}

// CleanupRun 清理指定 run。
//
// 对于 overwrite 模式，等同于 Cleanup。
func (s *Store) CleanupRun(runID string) error {
	return s.impl.CleanupRun(runID)
}

// Cleanup 清理整个 session（包括所有 run 目录）。
func (s *Store) Cleanup() error {
	return s.impl.Cleanup()
}
