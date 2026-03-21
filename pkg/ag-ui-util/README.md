# AG-UI Util

AG-UI 协议事件转换，将不同来源的事件流转换为 AG-UI 格式。

## 转换器

| 转换器 | 输入 | 使用场景 |
|--------|------|---------|
| OpencodeTranslator | opencode JSON 字符串 | wga-sandbox 输出转换 |
| EinoTranslator | eino AgentEvent | wga.Run() 输出转换（单智能体） |
| EinoMultiAgentTranslator | eino AgentEvent | wga.Run() 输出转换（多智能体，ActivityDelta 模式） |
| EinoMultiAgentSimpleTranslator | eino AgentEvent | wga.Run() 输出转换（多智能体，直接事件模式） |

## 使用

```go
import ag_ui_util "github.com/UnicomAI/wanwu/pkg/ag-ui-util"

// OpencodeTranslator - wga-sandbox 输出转换
runSession, outputCh, _ := wga_sandbox.Run(ctx, opts...)
tr := ag_ui_util.NewOpencodeTranslator(runSession.RunID, runSession.ThreadID)
eventCh := tr.TranslateStream(ctx, outputCh)
jsonCh := ag_ui_util.EventsToJSONChannel(ctx, eventCh)

// EinoTranslator - wga 单智能体输出转换
runSession, iter, _ := wga.Run(ctx, agentID, opts...)
tr := ag_ui_util.NewEinoTranslator(runSession.RunID, runSession.ThreadID)
eventCh := tr.TranslateStream(ctx, iter)
jsonCh := ag_ui_util.EventsToJSONChannel(ctx, eventCh)

// EinoMultiAgentTranslator - wga 多智能体输出转换
// 使用 State Management Events (StateSnapshot/StateDelta) 区分不同智能体的消息
runSession, iter, _ := wga.Run(ctx, agentID, opts...)
tr := ag_ui_util.NewEinoMultiAgentTranslator(runSession.RunID, runSession.ThreadID)
eventCh := tr.TranslateStream(ctx, iter)
jsonCh := ag_ui_util.EventsToJSONChannel(ctx, eventCh)

// EinoMultiAgentSimpleTranslator - wga 多智能体输出转换（简化模式）
// 直接发送独立事件，不使用 ActivityDelta 封装
runSession, iter, _ := wga.Run(ctx, agentID, opts...)
tr := ag_ui_util.NewEinoMultiAgentSimpleTranslator(runSession.RunID, runSession.ThreadID)
eventCh := tr.TranslateStream(ctx, iter)
jsonCh := ag_ui_util.EventsToJSONChannel(ctx, eventCh)
```

## 多智能体模式

多智能体模式通过 AG-UI 协议的 State Management Events 来区分不同智能体的消息：

- **StateSnapshot**: 包含当前活跃智能体 ID、所有智能体 ID 列表和名称映射
- **StateDelta**: 当智能体切换时发送，使用 JSON Patch 格式更新 `currentAgentId`

### 状态结构

```json
{
  "currentAgentId": "agent-1",
  "agentIds": ["supervisor", "agent-1", "agent-2"],
  "agentNames": {
    "supervisor": "supervisor",
    "agent-1": "agent-1",
    "agent-2": "agent-2"
  }
}
```

### 智能体切换流程

1. 检测到 `AgentEvent.AgentName` 变化
2. 发送前一个智能体的消息结束事件（TextMessageEnd 等）
3. 发送 `StateDelta` 事件更新 `currentAgentId`
4. 为新智能体创建独立的消息 ID
5. 发送新智能体的消息内容

## Reasoning 处理

转换器自动处理 reasoning（思考过程）的 Markdown 引用格式：

```
> 💭
> 思考内容第一行
> 思考内容第二行

正常回复内容...
```

## 多智能体转换器对比

| 特性 | EinoMultiAgentTranslator | EinoMultiAgentSimpleTranslator |
|------|-------------------------|--------------------------------|
| 事件封装 | 使用 ActivityDelta 封装每个事件 | 直接发送独立事件 |
| 事件结构 | `{"type":"ACTIVITY_DELTA","value":{...}}` | `{"type":"TEXT_MESSAGE_START",...}` |
| 状态管理 | 支持 StateSnapshot/StateDelta | 无状态管理 |
| 适用场景 | 需要前端状态同步 | 简化流程，直接消费事件 |
| 智能体切换 | 通过 ActivityDelta 更新 currentAgentId | 通过 ActivitySnapshot 切换 |

### EinoMultiAgentTranslator (ActivityDelta 模式)

所有事件都封装在 `ActivityDelta` 中，适合需要前端维护完整状态树的场景：

```json
{"type":"ACTIVITY_DELTA","value":{"messageId":"xxx","activityType":"agent_activity","delta":[{"type":"TEXT_MESSAGE_START",...}]}}
```

### EinoMultiAgentSimpleTranslator (直接事件模式)

直接发送独立事件，更简洁，适合不需要状态管理的场景：

```json
{"type":"TEXT_MESSAGE_START","messageId":"xxx","role":"assistant"}
```

## API

### 转换器

| 函数 | 说明 |
|------|------|
| `NewOpencodeTranslator(runID, threadID)` | 创建 opencode 转换器 |
| `NewEinoTranslator(runID, threadID)` | 创建 eino 单智能体转换器 |
| `NewEinoMultiAgentTranslator(runID, threadID)` | 创建 eino 多智能体转换器（ActivityDelta 模式） |
| `NewEinoMultiAgentSimpleTranslator(runID, threadID)` | 创建 eino 多智能体转换器（直接事件模式） |

### OpencodeTranslator

| 方法 | 说明 |
|------|------|
| `TranslateStream(ctx, <-chan string)` | 转换 opencode JSON 字符串流 |

### EinoTranslator

| 方法 | 说明 |
|------|------|
| `TranslateStream(ctx, *adk.AsyncIterator[*adk.AgentEvent])` | 转换 eino AgentEvent 迭代器 |

### 辅助函数

| 函数 | 说明 |
|------|------|
| `EventsToJSONChannel(ctx, events)` | 事件流 → JSON 字符串流 |
| `RemoveReasoningContent(content)` | 移除文本中的 reasoning 内容 |
