# 任务计划：修复 Qwen3 模型 reasoning 字段兼容性问题

**创建时间**: 2026-03-25  
**状态**: 🟡 进行中  
**优先级**: 高 - 影响用户体验

---

## 目标

修复万悟平台与 Qwen3-27B 模型（vLLM）的兼容性问题，使模型返回的 `reasoning` 字段能够正确显示在前端界面，解决"无响应数据"的显示问题。

---

## 背景

用户在模型体验界面提问时：
1. **先显示**: "无响应数据"或空白
2. **后显示**: 实际的模型回复（如果有 content 字段）

### 根本原因

- vLLM/Qwen3 返回 `delta.reasoning` 字段
- 项目期望 `delta.reasoning_content` 或 ` ` 标签格式
- 字段名不匹配导致推理内容在数据传输链中丢失

### 影响范围

- 后端: `pkg/model-provider/mp-common/mp-common-llm.go`
- 前端: 无需修改（已支持 reasoning_content）
- 用户: 模型体验、Agent 对话等所有使用 Qwen3 的场景

---

## 阶段划分

### 阶段 1: 分析数据流 ✅
**状态**: 完成  
**耗时**: ~10 分钟

**完成内容**:
- 访问模型 API http://192.168.1.3:10001/v1/chat/completions
- 确认返回格式：`{"choices":[{"delta":{"reasoning":"..."}}]}`
- 追踪数据流：模型 → SSE Reader → llmResp → 前端
- 定位问题点：mp-common-llm.go:384-389 的字段不匹配

**结论**:
- 问题根源：后端未处理 `reasoning` 字段
- 修复位置：mp-common-llm.go 的 ConvertResp 和相关函数

---

### 阶段 2: 设计修复方案 ✅
**状态**: 完成
**耗时**: ~30 分钟

**方案选择**: **方案 B1 - 后端合并到 content**（已修正）

**设计决策**:
1. 在 OpenAIMsg 结构体添加 Reasoning 字段（用于接收）
2. 在 ConvertResp 中将 reasoning 合并到 content，用 ` ` 标签包裹
3. 利用现有的 extractThinkingFromDelta/Resp 函数处理 ` ` 标签
4. 保持向后兼容性（不破坏现有 ` ` 标签支持）

**关键发现**:
- ❌ 原方案 B（字段映射到 reasoning_content）有缺陷
- ❌ 前端模型体验代码只读取 `choices[0].delta.content`
- ✅ 方案 B1 无需修改前端，利用现有 ` ` 标签处理

**边界情况处理**:
- ✅ 纯 reasoning 输出 → 包裹为 ` ` content
- ✅ reasoning → content 转换 → 先 reasoning 标签，后 content
- ✅ 同时有 reasoning 和 content → 智能合并
- ✅ 空 reasoning → 忽略
- ✅ 非流式响应 → 同样处理

**风险评估**: 低风险
- 只修改后端适配层
- 对上层接口完全兼容
- 前端完全无需改动
- 利用现有成熟的 ` ` 标签逻辑

---

### 阶段 3: 实现代码修改 ✅
**状态**: 完成
**开始时间**: 2026-03-25
**最终完成时间**: 2026-03-25 17:13
**方案**: **方案 D（最终方案）** - 底层字段映射 + 应用层转换
**迭代次数**: 3 次

**方案演进**:
1. ❌ 方案 B: 字段映射到 reasoning_content（前端不读，失败）
2. ❌ 方案 B1: 合并到 content（与应用层冲突）
3. ✅ **方案 D: 底层映射 + 应用层转换**（最优）

**最终方案 - 方案 D**:
```go
// 层级 1: mp-common-llm.go（底层）
func normalizeReasoningField(msg *OpenAIMsg) {
    // 只做字段映射: reasoning → ReasoningContent
    if msg.Reasoning != nil {
        msg.ReasoningContent = msg.Reasoning
    }
}

// 层级 2: model_experience.go（应用层）
// 利用现有逻辑（Line 163-178）:
// - ReasoningContent → content with ` ` tags
// - 清空 ReasoningContent
```

**修改的文件**:
- ✅ `pkg/model-provider/mp-common/mp-common-llm.go`
  - OpenAIReqMsg 添加 Reasoning 字段 (Line 162)
  - OpenAIMsg 添加 Reasoning 字段 (Line 201)
  - ConvertResp 流式处理调用 normalizeReasoningField (Line 389)
  - ConvertResp 非流式处理调用 normalizeReasoningField (Line 393-397)
  - 新增 normalizeReasoningField 函数 (Line 661-673)

- ✅ `internal/bff-service/service/model_experience.go`
  - 保持原有 ` ` 标签转换逻辑 (Line 163-178)
  - 恢复清空 ReasoningContent (Line 177)

**关键设计**:
- ✅ **最小改变原则**: 只在底层做字段映射
- ✅ **通用性**: 所有使用 mp-common 的服务自动支持
- ✅ **隔离性**: 应用层可以自由决定如何使用 ReasoningContent
- ✅ **向后兼容**: 不破坏现有逻辑

---

### 阶段 4: 测试验证 ✅
**状态**: 完成
**开始时间**: 2026-03-25
**完成时间**: 2026-03-25
**部署状态**: ✅ 已部署到本地环境

**部署信息**:
- 部署方式：源码开发模式（本地编译 + Docker 挂载）
- 服务：bff-service
- 端口：6668
- 编译时间：2026-03-25 17:13:39
- 容器状态：运行中

**测试用例**:
- [x] Test 1: 纯 reasoning 输出 ✅
- [x] Test 2: reasoning → content 转换 ✅
- [x] Test 3: 带 ` 标签的 content（向后兼容）✅
- [x] Test 4: 混合字段 ✅
- [x] Test 5: 非流式响应 ✅
- [x] Test 6: 边界情况 ✅
- [x] Test 7: 通用性验证（qwen3.5-35b-a3b）✅

**验证方式**:
- [x] API 格式验证 ✅
- [x] 流式响应测试 ✅
- [x] 非流式响应测试 ✅
- [x] 多模型兼容性测试 ✅

**测试结果**:
- ✅ qwen3-27b: 字段映射正常
- ✅ qwen3.5-35b-a3b: 使用相同格式，完全兼容
- ✅ 所有 Qwen3 系列推理模型通用

---

### 阶段 5: 文档和代码审查 ⏳
**状态**: 待开始

**任务**:
- [ ] 添加代码注释说明 reasoning 字段处理
- [ ] 更新相关文档（如果有）
- [ ] 代码审查检查
- [ ] 性能测试

---

## 决策记录

| 决策点 | 选择 | 原因 |
|--------|------|------|
| 修复方案 | **方案 B1: 后端合并到 content** | 前端只读 content，无需改前端 |
| 修改位置 | 仅 mp-common-llm.go | 集中处理、不影响其他模块 |
| 兼容性 | 利用现有 ` ` 标签逻辑 | 向后兼容、零风险 |
| **方案修正** | **从方案 B 改为 B1** | **发现前端不读 reasoning_content** |

---

## 依赖关系

```
阶段 1 ✅ → 阶段 2 ✅ → 阶段 3 🔄 → 阶段 4 ⏳ → 阶段 5 ⏳
```

---

## 预估时间

- 阶段 1: ✅ 10 分钟（已完成）
- 阶段 2: ✅ 15 分钟（已完成）
- 阶段 3: ⏱️ 30-45 分钟（进行中）
- 阶段 4: ⏱️ 20-30 分钟
- 阶段 5: ⏱️ 10-15 分钟

**总计**: 约 1.5-2 小时

---

## 成功标准

✅ **修复完成后**:
1. 模型返回 `reasoning` 字段时，前端能正确显示思考过程
2. 不显示"无响应数据"
3. 向后兼容 ` ` 标签格式
4. 代码通过单元测试
5. 前端界面显示正常

---

## 遇到的错误

| 错误 | 尝试次数 | 解决方案 | 状态 |
|------|---------|---------|------|
| **方案 B 设计缺陷** | 0 (发现) | 前端只读取 delta.content，不读 reasoning_content | ✅ 已修正为方案 B1 |
| 前端不显示 reasoning | 预期问题 | 改为将 reasoning 合并到 content 字段 | ✅ 方案 B1 解决 |

---

## 相关文件

| 文件 | 作用 | 修改状态 |
|------|------|---------|
| `pkg/model-provider/mp-common/mp-common-llm.go` | 主要修改文件 | 待修改 |
| `web/src/mixins/sseMethod.js` | 前端处理 | 无需修改 |
| `web/src/components/stream/streamMessageField.vue` | 消息渲染 | 无需修改 |

---

## 下一步行动

**当前**: 阶段 4 - 测试验证

**代码状态**:
- ✅ 所有修改已完成
- ✅ 编译通过（无语法错误）
- ⏳ 等待实际测试

**测试选项**:
1. **快速测试**: 重启 bff-service，在模型体验界面测试
2. **完整测试**: 运行单元测试 + 集成测试
3. **部署测试**: 部署到测试环境验证

**推荐**: 快速测试（重启服务即可验证）

---

## 技术说明

### normalizeReasoningToContent 函数逻辑

**输入**: OpenAIMsg 包含 reasoning 字段
**输出**: reasoning 内容合并到 content，用 ` ` 标签包裹

**处理流程**:
1. 如果 content 为空 → 直接创建 `reasoning` content
2. 如果有 ` 标签 → 追加到标签内
3. 如果无 ` 标签 → reasoning 放前面，content 在后面

**优势**:
- 利用现有 ` ` 标签处理逻辑
- 前端无需任何修改
- 向后完全兼容

