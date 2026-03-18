package persistent

import (
	"path/filepath"
	"sync"

	"github.com/UnicomAI/wanwu/pkg/wga-persistent/internal/storage"
)

// overwritePersistent overwrite 模式实现。
type overwritePersistent struct {
	storage  storage.Storage
	threadID string
	mu       sync.RWMutex
}

func newOverwritePersistent(s storage.Storage, threadID string) SessionPersistent {
	return &overwritePersistent{
		storage:  s,
		threadID: threadID,
	}
}

func (p *overwritePersistent) GetThreadDir() SessionDirInfo {
	threadDir := buildThreadDir(p.threadID, ModeOverwrite)
	return SessionDirInfo{
		ThreadID:  p.threadID,
		RunID:     "",
		Mode:      ModeOverwrite,
		Dir:       filepath.Join(p.storage.BaseDir(), threadDir),
		Timestamp: 0,
	}
}

func (p *overwritePersistent) GetRunDir(_ string, opts ...Option) (bool, SessionDirInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	info := p.GetThreadDir()
	ok, err := p.storage.Exists(info.Dir)
	if err != nil {
		return false, info, err
	}

	cfg := applyOptions(opts...)
	if cfg.mkdir && !ok {
		if err := p.storage.Mkdir(info.Dir, cfg.perm); err != nil {
			return false, info, err
		}
		return true, info, nil
	}

	return ok, info, nil
}

func (p *overwritePersistent) GetLastRunDir() (bool, SessionDirInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	info := p.GetThreadDir()
	ok, err := p.storage.Exists(info.Dir)
	if err != nil {
		return false, info, err
	}
	return ok, info, nil
}

func (p *overwritePersistent) ListRunDirs() ([]SessionDirInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	info := p.GetThreadDir()
	ok, err := p.storage.Exists(info.Dir)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return []SessionDirInfo{info}, nil
}

func (p *overwritePersistent) CleanupRun(_ string) error {
	return p.Cleanup()
}

func (p *overwritePersistent) Cleanup() error {
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
