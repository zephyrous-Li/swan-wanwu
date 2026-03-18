package persistent

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/UnicomAI/wanwu/pkg/wga-persistent/internal/storage"
)

// versionedPersistent versioned 模式实现。
type versionedPersistent struct {
	storage  storage.Storage
	threadID string
	mu       sync.RWMutex
}

func newVersionedPersistent(s storage.Storage, threadID string) SessionPersistent {
	return &versionedPersistent{
		storage:  s,
		threadID: threadID,
	}
}

func (p *versionedPersistent) GetThreadDir() SessionDirInfo {
	threadDir := buildThreadDir(p.threadID, ModeVersioned)
	return SessionDirInfo{
		ThreadID:  p.threadID,
		RunID:     "",
		Mode:      ModeVersioned,
		Dir:       filepath.Join(p.storage.BaseDir(), threadDir),
		Timestamp: 0,
	}
}

func (p *versionedPersistent) GetRunDir(runID string, opts ...Option) (bool, SessionDirInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	threadInfo := p.GetThreadDir()

	// 查找已存在的 run 目录
	dirs, err := p.storage.ListDirs(threadInfo.Dir)
	if err != nil {
		return false, SessionDirInfo{}, err
	}
	for _, dir := range dirs {
		id, timestamp, ok := parseRunDir(dir)
		if ok && id == runID {
			info := SessionDirInfo{
				ThreadID:  p.threadID,
				RunID:     runID,
				Mode:      ModeVersioned,
				Dir:       filepath.Join(threadInfo.Dir, dir),
				Timestamp: timestamp,
			}
			return true, info, nil
		}
	}

	// 不存在，生成新目录
	runDir := buildRunDir(runID)
	info := SessionDirInfo{
		ThreadID:  p.threadID,
		RunID:     runID,
		Mode:      ModeVersioned,
		Dir:       filepath.Join(threadInfo.Dir, runDir),
		Timestamp: time.Now().UnixMilli(),
	}

	cfg := applyOptions(opts...)
	if cfg.mkdir {
		// 如果需要复制，先获取上一次的 run 目录（在创建新目录之前）
		var lastRunDir string
		if cfg.copyLastOutput {
			lastRunDir = p.getLastRunDirLocked(dirs, threadInfo.Dir)
		}

		if err := p.storage.Mkdir(info.Dir, cfg.perm); err != nil {
			return false, info, err
		}

		// 复制上一次的输出到当前目录
		if lastRunDir != "" && lastRunDir != info.Dir {
			if err := copyDir(lastRunDir, info.Dir); err != nil {
				return false, info, err
			}
		}

		return true, info, nil
	}

	return false, info, nil
}

// getLastRunDirLocked 获取最新的 run 目录（假设已持有锁）。
func (p *versionedPersistent) getLastRunDirLocked(dirs []string, threadDir string) string {
	if len(dirs) == 0 {
		return ""
	}

	type runDirWithInfo struct {
		name      string
		timestamp int64
	}
	var runDirs []runDirWithInfo
	for _, dir := range dirs {
		_, timestamp, ok := parseRunDir(dir)
		if ok {
			runDirs = append(runDirs, runDirWithInfo{
				name:      dir,
				timestamp: timestamp,
			})
		}
	}

	if len(runDirs) == 0 {
		return ""
	}

	sort.Slice(runDirs, func(i, j int) bool {
		return runDirs[i].timestamp > runDirs[j].timestamp
	})

	return filepath.Join(threadDir, runDirs[0].name)
}

func (p *versionedPersistent) GetLastRunDir() (bool, SessionDirInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	threadInfo := p.GetThreadDir()
	ok, err := p.storage.Exists(threadInfo.Dir)
	if err != nil {
		return false, threadInfo, err
	}
	if !ok {
		return false, SessionDirInfo{}, nil
	}

	dirs, err := p.storage.ListDirs(threadInfo.Dir)
	if err != nil || len(dirs) == 0 {
		return false, SessionDirInfo{}, nil
	}

	type runDirWithInfo struct {
		name      string
		runID     string
		timestamp int64
	}
	var runDirs []runDirWithInfo
	for _, dir := range dirs {
		runID, timestamp, ok := parseRunDir(dir)
		if ok {
			runDirs = append(runDirs, runDirWithInfo{
				name:      dir,
				runID:     runID,
				timestamp: timestamp,
			})
		}
	}

	if len(runDirs) == 0 {
		return false, SessionDirInfo{}, nil
	}

	sort.Slice(runDirs, func(i, j int) bool {
		return runDirs[i].timestamp > runDirs[j].timestamp
	})

	latest := runDirs[0]
	info := SessionDirInfo{
		ThreadID:  p.threadID,
		RunID:     latest.runID,
		Mode:      ModeVersioned,
		Dir:       filepath.Join(threadInfo.Dir, latest.name),
		Timestamp: latest.timestamp,
	}
	return true, info, nil
}

func (p *versionedPersistent) ListRunDirs() ([]SessionDirInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	threadInfo := p.GetThreadDir()
	ok, err := p.storage.Exists(threadInfo.Dir)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	dirs, err := p.storage.ListDirs(threadInfo.Dir)
	if err != nil {
		return nil, err
	}

	var result []SessionDirInfo
	for _, dir := range dirs {
		runID, timestamp, ok := parseRunDir(dir)
		if ok {
			result = append(result, SessionDirInfo{
				ThreadID:  p.threadID,
				RunID:     runID,
				Mode:      ModeVersioned,
				Dir:       filepath.Join(threadInfo.Dir, dir),
				Timestamp: timestamp,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp > result[j].Timestamp
	})

	return result, nil
}

func (p *versionedPersistent) CleanupRun(runID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	threadInfo := p.GetThreadDir()
	dirs, err := p.storage.ListDirs(threadInfo.Dir)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		id, _, ok := parseRunDir(dir)
		if ok && id == runID {
			return p.storage.Remove(filepath.Join(threadInfo.Dir, dir))
		}
	}

	return nil
}

func (p *versionedPersistent) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	info := p.GetThreadDir()
	ok, err := p.storage.Exists(info.Dir)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return p.storage.Remove(info.Dir)
}

// ============================================================================
// 辅助函数 - versioned 专用
// ============================================================================

const (
	runDirPrefix = "run-"
)

// buildRunDir 构建 run 目录名：run-{timestamp}_{runID}
func buildRunDir(runID string) string {
	return runDirPrefix + strconv.FormatInt(time.Now().UnixMilli(), 10) + dirSeparator + runID
}

// parseRunDir 解析 run 目录名。
func parseRunDir(name string) (runID string, timestamp int64, ok bool) {
	if !strings.HasPrefix(name, runDirPrefix) {
		return "", 0, false
	}
	rest := strings.TrimPrefix(name, runDirPrefix)
	parts := strings.SplitN(rest, dirSeparator, 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return parts[1], ts, true
}
