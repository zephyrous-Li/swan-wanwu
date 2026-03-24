package ag_ui_util

import (
	"context"
	"io"

	"github.com/UnicomAI/wanwu/pkg/util"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// EinoTranslator 将 eino AgentEvent 转换为 AG-UI 事件，用于单智能体场景。
//
// 转换规则：
//   - 每个运行开始时发送 RUN_STARTED
//   - Assistant 消息转换为 TEXT_MESSAGE_START/CONTENT/END 序列
//   - ReasoningContent 转换为 REASONING_START/END 和 REASONING_MESSAGE_START/CONTENT/END 序列
//   - ToolCalls 转换为 TOOL_CALL_START/ARGS/END 序列
//   - Tool 消息（结果）转换为 TOOL_CALL_RESULT
//   - 运行结束时发送 RUN_FINISHED
//
// AG-UI 协议要求：
//   - Tool 消息处理后需要重置消息状态，后续 Assistant 响应使用新的 messageId
//   - Reasoning 和 TextMessage 是独立的消息流，各自有独立的 ID
//   - ToolCall 通过 parentMessageId 关联到所属的 Assistant 消息
//   - TEXT_MESSAGE 和 TOOL_CALL 可以穿插，但 REASONING 只能顺序进行
type EinoTranslator struct {
	BaseState
	toolCallIDs map[string]bool
}

// NewEinoTranslator 创建 eino 转换器。
func NewEinoTranslator(threadID, runID string) *EinoTranslator {
	return &EinoTranslator{
		BaseState:   NewBaseState(threadID, runID),
		toolCallIDs: make(map[string]bool),
	}
}

func (t *EinoTranslator) TranslateStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) <-chan aguievents.Event {
	out := make(chan aguievents.Event, 1024)
	go func() {
		defer util.PrintPanicStack()
		defer close(out)
		defer func() {
			for _, evt := range t.FinishBase() {
				select {
				case out <- evt:
				case <-ctx.Done():
					return
				}
			}
		}()

		for {
			event, ok := iter.Next()
			if !ok {
				return
			}

			if event.Err != nil {
				errMsg := &schema.Message{
					Role:    schema.Assistant,
					Content: "[error] " + event.Err.Error(),
				}
				for _, evt := range t.translateMessage(errMsg) {
					select {
					case out <- evt:
					case <-ctx.Done():
						return
					}
				}
				return
			}

			if event.Action != nil && event.Action.Exit {
				return
			}

			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}

			if t.MessageID() == "" {
				t.SetMessageID(aguievents.GenerateMessageID())
			}

			msgOutput := event.Output.MessageOutput

			if msgOutput.IsStreaming {
				t.translateStream(ctx, msgOutput, out)
			} else {
				for _, evt := range t.translateMessage(msgOutput.Message) {
					select {
					case out <- evt:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out
}

func (t *EinoTranslator) translateStream(ctx context.Context, msgOutput *adk.MessageVariant, out chan<- aguievents.Event) {
	if msgOutput.MessageStream == nil {
		return
	}
	defer msgOutput.MessageStream.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		frame, err := msgOutput.MessageStream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			return
		}

		for _, evt := range t.translateMessage(frame) {
			select {
			case out <- evt:
			case <-ctx.Done():
				return
			}
		}
	}
}

// translateMessage 转换 eino Message 为 AG-UI 事件。
//
// AG-UI 协议映射：
//  1. Tool 消息（Role=Tool）：
//     - 调用 EndAll() 关闭所有活跃的消息
//     - 发送 TOOL_CALL_RESULT（使用新的 messageId）
//     - 重置消息状态，为后续 Assistant 消息准备
//  2. ToolCalls：
//     - 调用 EndAll() 关闭 Reasoning 和 TextMessage
//     - 发送 TOOL_CALL_START/ARGS/END，parentMessageId 关联当前消息
//  3. ReasoningContent：
//     - 关闭 TextMessage
//     - 发送 REASONING_START → REASONING_MESSAGE_START → CONTENT
//  4. Content：
//     - 关闭 Reasoning（REASONING_MESSAGE_END → REASONING_END）
//     - 发送 TEXT_MESSAGE_START → CONTENT
//
// 事件顺序符合 AG-UI 规范：
//   - REASONING 只能顺序：START → MESSAGE_START → CONTENT → MESSAGE_END → END
//   - TEXT_MESSAGE 和 TOOL_CALL 可以穿插
func (t *EinoTranslator) translateMessage(msg *schema.Message) []aguievents.Event {
	if msg == nil {
		return nil
	}

	// 处理工具调用结果
	if msg.Role == schema.Tool && msg.ToolCallID != "" {
		var events []aguievents.Event
		events = append(events, t.EnsureRunStarted()...)
		events = append(events, t.EndAll()...)
		events = append(events, aguievents.NewToolCallResultEvent(aguievents.GenerateMessageID(), msg.ToolCallID, msg.Content))
		t.ResetMessageID()
		return events
	}

	hasContent := msg.Content != "" || msg.ReasoningContent != "" || len(msg.ToolCalls) > 0
	if !hasContent {
		return nil
	}

	var events []aguievents.Event
	events = append(events, t.EnsureRunStarted()...)

	if len(msg.ToolCalls) > 0 {
		parentMsgID := t.MessageID()
		events = append(events, t.EndAll()...)

		for _, tc := range msg.ToolCalls {
			if tc.ID == "" || tc.Function.Name == "" {
				continue
			}
			if !t.toolCallIDs[tc.ID] {
				events = append(events, aguievents.NewToolCallStartEvent(tc.ID, tc.Function.Name, aguievents.WithParentMessageID(parentMsgID)))
				if tc.Function.Arguments != "" {
					events = append(events, aguievents.NewToolCallArgsEvent(tc.ID, tc.Function.Arguments))
				}
				events = append(events, aguievents.NewToolCallEndEvent(tc.ID))
				t.toolCallIDs[tc.ID] = true
			}
		}
	}

	if msg.ReasoningContent != "" {
		events = append(events, t.EndTextMessage()...)
		events = append(events, t.StartReasoning()...)
		events = append(events, t.StartReasoningMessage()...)
		events = append(events, aguievents.NewReasoningMessageContentEvent(t.ReasoningMessageID(), msg.ReasoningContent))
	}

	if msg.Content != "" {
		events = append(events, t.EndReasoningMessage()...)
		events = append(events, t.EndReasoning()...)
		events = append(events, t.StartTextMessage()...)
		events = append(events, aguievents.NewTextMessageContentEvent(t.MessageID(), msg.Content))
	}

	return events
}
