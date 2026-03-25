# 研究发现：Qwen3 模型 reasoning 字段问题

**更新时间**: 2026-03-25

---

## 模型 API 输出格式

### vLLM/Qwen3 实际返回

```json
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","model":"qwen3-27b","choices":[{"index":0,"delta":{"reasoning":"Thinking Process:\n\n1. Analyze the Request\n\n2. Identify the Core Task\n\n..."}}]}
```

**关键特征**:
- 字段名: `delta.reasoning`（非 `reasoning_content`）
- 只返回推理过程，`content` 字段为空
- 使用 SSE 流式格式

### 非流式返回

```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": null,
      "reasoning": "Thinking Process:\n\n1. Analyze..."
    }
  }]
}
```

---

## 数据流追踪

### 完整数据链路

```
1. 模型 API (vLLM)
   ↓ 返回: delta.reasoning
   
2. SSE Reader (sse_util/sse_reader.go)
   ↓ 读取原始 SSE 行
   
3. llmResp.ConvertResp() (mp-common-llm.go:356-405)
   ↓ ❌ 问题点：未处理 reasoning 字段
   ↓ ❌ extractThinkingFromDelta 只检查 Content
   
4. 前端接收 (sseMethod.js:380-407)
   ↓ 接收: data.data.reasoning_content (为空)
   
5. 前端渲染 (streamMessageField.vue:1118-1120)
   ↓ 检查: if (!data.response && finish !== 0)
   ↓ 显示: "无响应数据"
```

---

## 问题分析

### 字段名不匹配

| 层级 | 期望字段 | 实际字段 | 结果 |
|------|---------|---------|------|
| 模型 | - | `reasoning` | ✅ 正常输出 |
| 后端 | `reasoning_content` | `reasoning` | ❌ 不匹配 |
| 前端 | `reasoning_content` | 空 | ❌ 无数据显示 |

### 代码分析

#### mp-common-llm.go:384-389 (流式处理)
```go
if resp.stream {
    if len(ret.Choices) > 0 && ret.Choices[0].Delta != nil {
        delta := ret.Choices[0].Delta
        if delta.Role == "" {
            delta.Role = MsgRoleAssistant
        }
        resp.inThinking, _ = extractThinkingFromDelta(delta, resp.inThinking)
        // ❌ 问题：没有先处理 reasoning 字段
    }
}
```

#### mp-common-llm.go:604-651 (extractThinkingFromDelta)
```go
func extractThinkingFromDelta(delta *OpenAIMsg, inThinking bool) (bool, *string) {
    // ...
    if delta.Content == "" {
        return inThinking, nil  // ❌ Content 为空时直接返回
    }
    // 只处理 ` 标签，不处理独立 reasoning 字段
    // ...
}
```

#### 前端 sseMethod.js:380-383
```javascript
const reasoning = data.data && data.data.reasoning_content
    ? data.data.reasoning_content
    : '';  // ❌ reasoning_content 为空（后端没传递）
```

#### 前端 streamMessageField.vue:1118-1120
```javascript
replaceLastData(index, data) {
    if (!data.response && data.finish !== 0) {
        data.response = this.$t('app.noResponse');  // "无响应数据"
    }
    // ...
}
```

---

## 用户症状

### 时间线

```
T+0s:   用户发送问题
        ↓
T+1s:   模型开始输出 reasoning
        ↓ 后端：字段不匹配，内容丢失
        ↓ 前端：显示 responseLoading=true
        ↓ 用户看到：空白或"思考中..."
        
T+5s:   推理完成，模型开始输出 content
        ↓ 后端：正确提取 content
        ↓ 前端：开始显示回复
        ↓ 用户看到：答案出现
        
T+10s:  流式结束 (finish: 1)
        ↓ 前端：检查 response 是否为空
        ↓ 如果为空：显示"无响应数据"
```

### 不同场景

| 场景 | 模型输出 | 用户看到 | 原因 |
|------|---------|---------|------|
| 场景 A | 只有 reasoning | "无响应数据" | 字段丢失 |
| 场景 B | reasoning → content | 先空白，后显示答案 | reasoning 丢失 |
| 场景 C | 只有 content | 正常显示 | 无影响 |
| 场景 D | 带 ` 标签 | 正常显示（思考可折叠） | 已支持 |

---

## 相关 OpenAI 标准

### OpenAI o1 推理格式

```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "最终答案",
      "reasoning_content": "思考过程..."
    }
  }]
}
```

### 项目支持的格式

1. **标准格式**: `reasoning_content` 字段
2. **标签格式**: ` ` 标签包裹
3. **不支持**: `reasoning` 字段 ← 需要添加

---

## 技术债务

### 现有代码的假设

- 假设 1: 所有模型都使用 `reasoning_content` 或 ` ` 标签
- 假设 2: 思考内容一定在 `content` 字段中
- 假设 3: 不存在独立的 `reasoning` 字段

### 需要修正

- 支持多种字段名（reasoning, reasoning_content）
- 处理独立字段（不在 content 中）
- 优先级处理（reasoning_content > reasoning）

---

## 参考资源

### 相关文件

- `pkg/model-provider/mp-common/mp-common-llm.go` - 模型适配层
- `web/src/mixins/sseMethod.js` - 前端 SSE 处理
- `web/src/components/stream/streamMessageField.vue` - 消息渲染

### 相关 RFC/标准

- OpenAI API: Chat Completions
- vLLM 文档: https://docs.vllm.ai/

---

## 待验证

- [x] 其他模型是否也使用 `reasoning` 字段？ ✅ **已验证：qwen3.5-35b-a3b 使用相同格式**
- [ ] Qwen3 是否有配置参数控制输出格式？
- [ ] vLLM 版本是否有影响？

---

## 通用性验证 ✅

### 测试模型：qwen3.5-35b-a3b

**测试时间**: 2026-03-25

**测试结果**:

#### 流式响应
```
delta.reasoning = "嗯"
delta.reasoning = "，"
delta.reasoning = "用户"
...
```

#### 非流式响应
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "\n\n1 + 1 = 2",
      "reasoning": "Thinking Process:\n\n1. Analyze the Request..."
    }
  }]
}
```

### 结论

✅ **完全通用** - qwen3.5-35b-a3b 使用与 qwen3-27b 完全相同的字段格式

**通用性范围**:
- 所有基于 vLLM 的 Qwen3 系列推理模型
- 流式和非流式响应
- 所有使用 `mp-common` 包的服务（bff、agent、assistant 等）

**数据流**:
```
vLLM (qwen3.x): delta.reasoning/message.reasoning
  ↓
mp-common-llm.go: normalizeReasoningField
  ↓ ReasoningContent
应用层 (model_experience.go): 转换为 content + 标签
  ↓
前端: 正确显示
```

---

## 关键发现：方案 B 不可行 ⚠️

### 问题发现时间
2026-03-25（方案评估阶段）

### 问题描述

原方案 B（字段映射到 reasoning_content）存在致命缺陷：
- 后端可以设置 `reasoning_content` 字段 ✅
- **但前端模型体验代码不读取该字段** ❌

### 证据

**前端代码** (sseMethod.js:915-927):
```javascript
if (Array.isArray(data.choices) && data.choices[0] && data.choices[0].delta) {
    data.response = data.choices[0].delta.content;  // 只读 content！
    // ...
}
```

### 结论

**必须采用方案 B1**：将 reasoning 合并到 content 字段
- 前端完全无需修改
- 利用现有 ` ` 标签处理逻辑
- 零风险，向后兼容

