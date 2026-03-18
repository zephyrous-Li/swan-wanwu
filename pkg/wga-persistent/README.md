# WGA Persistent

会话持久化存储管理，支持多轮对话中生成的文件持久化存储。

## 架构

```
pkg/wga-persistent/
├── persistent.go                    # 对外 API
│   ├── Mode (类型别名)
│   ├── Option (类型别名)
│   ├── SessionDirInfo (类型别名)
│   └── Store
│
├── persistent_test.go               # 测试
│
├── README.md
│
└── internal/
    ├── storage/                     # 存储后端
    │   └── storage.go
    │       └── Storage (接口)
    │       └── localStorage (实现)
    │
    └── persistent/                  # 持久化布局
        ├── persistent.go
        │   ├── Mode, SessionDirInfo, Option
        │   └── SessionPersistent (接口)
        ├── persistent_overwrite.go
        │   └── overwritePersistent
        └── persistent_versioned.go
            └── versionedPersistent
```

## 目录结构

### overwrite 模式

```
{baseDir}/
└── thread-overwrite_{threadID}/
    ├── file1.txt        ← 直接放文件
    └── file2.txt        ← 每次覆盖
```

### versioned 模式

```
{baseDir}/
└── thread-versioned_{threadID}/
    ├── run-{timestamp1}_{runID1}/
    │   ├── file1.txt
    │   └── file2.txt
    └── run-{timestamp2}_{runID2}/
        ├── file1.txt    ← 修改后的版本
        └── file2.txt
```

## 使用

```go
import wga_persistent "github.com/UnicomAI/wanwu/pkg/wga-persistent"

// 创建 Store（绑定 session）
store, err := wga_persistent.NewStore(
    wga_persistent.ModeVersioned,
    "/data/persistent",
    "thread-123",
)

// 获取恢复目录（最新版本）
ok, restoreInfo, err := store.GetLastRunDir()
if ok {
    // 从 restoreInfo.Dir 恢复文件
}

// 获取保存目录（不创建）
ok, saveInfo, err := store.GetRunDir("run-456")
// ok=false: 目录不存在
// ok=true: 目录已存在

// 获取保存目录（自动创建）
ok, saveInfo, err := store.GetRunDir("run-456", wga_persistent.WithMkdir(false))
// ok=true: 目录存在（无论是否新创建）
// 将文件保存到 saveInfo.Dir

// 获取保存目录（自动创建并从上一次输出复制）
ok, saveInfo, err := store.GetRunDir("run-456", wga_persistent.WithMkdir(true))
// ok=true: 目录存在，已从最新 run 目录复制内容

// 自定义权限
ok, saveInfo, err := store.GetRunDir("run-456", wga_persistent.WithMkdir(false, 0700))

// 列出所有 run
dirs, err := store.ListRunDirs()
for _, dir := range dirs {
    fmt.Printf("RunID: %s, Timestamp: %d, Dir: %s\n", dir.RunID, dir.Timestamp, dir.Dir)
}

// 清理指定 run
store.CleanupRun("run-456")

// 清理整个会话
store.Cleanup()
```

## 与 wga-sandbox 集成示例

```go
// 创建持久化存储
store, err := wga_persistent.NewStore(
    wga_persistent.ModeVersioned,
    "/data/persistent",
    threadID,
)

// 准备阶段：恢复
ok, restoreInfo, err := store.GetLastRunDir()
if ok {
    sb.CopyToSandbox(ctx, restoreInfo.Dir)
}

// 执行任务...

// 完成阶段：保存（自动创建目录）
_, saveInfo, err := store.GetRunDir(runID, wga_persistent.WithMkdir(false))
sb.CopyFromSandbox(ctx, saveInfo.Dir)
```

## 并发安全

Store 是并发安全的：
- 内部使用读写锁保护共享状态
- `GetRunDir` 使用写锁保证原子性

使用 `WithMkdir` 确保并发场景下对同一 runID 只创建一个目录：

```go
// 多个 goroutine 并发调用，返回相同目录
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        ok, info, _ := store.GetRunDir("run-1", wga_persistent.WithMkdir(false))
        // ok=true, info.Dir 相同
    }()
}
wg.Wait()
```

## API

### Store

| 方法 | 说明 |
|------|------|
| `NewStore(mode, baseDir, threadID)` | 创建 Store，目录已存在时检测 mode 冲突 |
| `GetThreadDir()` | 获取 session 目录信息 |
| `GetRunDir(runID, opts...)` | 获取指定 run 的保存目录 |
| `GetLastRunDir()` | 获取最新的恢复目录 |
| `ListRunDirs()` | 列出所有 run 目录 |
| `CleanupRun(runID)` | 清理指定 run |
| `Cleanup()` | 清理整个 session |

### GetRunDir 返回值

```go
ok, info, err := store.GetRunDir(runID, opts...)
```

| 场景 | ok | 说明 |
|------|-----|------|
| 目录不存在，无 `WithMkdir` | `false` | 返回路径信息，需手动创建 |
| 目录不存在，有 `WithMkdir` | `true` | 自动创建目录 |
| 目录已存在 | `true` | 返回已存在目录信息 |

### SessionDirInfo

| 字段 | 类型 | 说明 |
|------|------|------|
| `ThreadID` | string | 会话 ID |
| `RunID` | string | 执行 ID（overwrite 模式为空） |
| `Mode` | Mode | 存储模式 |
| `Dir` | string | 完整路径 |
| `Timestamp` | int64 | 毫秒时间戳（overwrite 模式为 0） |

### Mode

| 值 | 说明 |
|------|------|
| `ModeOverwrite` | 覆盖模式：每次执行覆盖同一目录 |
| `ModeVersioned` | 分轮存储模式：每次执行创建独立的 run 目录 |

### Option

| 函数 | 说明 |
|------|------|
| `WithMkdir(copyLastOutput)` | 创建目录（默认权限 0755），copyLastOutput 是否从最新 run 复制内容 |
| `WithMkdir(copyLastOutput, perm)` | 创建目录（自定义权限），copyLastOutput 是否从最新 run 复制内容 |

**copyLastOutput 参数说明**：
- `false`: 仅创建空目录
- `true`: 创建目录并从最新的 run 目录复制内容（仅 versioned 模式有效，overwrite 模式忽略此参数）
