// Package ag_ui_util 提供 AG-UI 协议的事件转换功能。
package ag_ui_util

import aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

// MessageState 管理 AG-UI 消息的状态。
// 跟踪文本消息和推理消息的活跃状态，确保事件的正确顺序。
//
// AG-UI 协议要求：
//   - TEXT_MESSAGE 必须以 START 开始，END 结束
//   - REASONING 必须以 START 开始，END 结束
//   - REASONING_MESSAGE 必须以 START 开始，END 结束
//   - 所有方法是幂等的，可安全多次调用
type MessageState struct {
	textMsgID           string
	reasoningID         string
	reasoningMsgID      string
	textStarted         bool
	reasoningStarted    bool
	reasoningMsgStarted bool
}

// NewMessageState 创建消息状态。
func NewMessageState() *MessageState {
	return &MessageState{}
}

func (s *MessageState) TextMsgID() string           { return s.textMsgID }
func (s *MessageState) ReasoningID() string         { return s.reasoningID }
func (s *MessageState) ReasoningMsgID() string      { return s.reasoningMsgID }
func (s *MessageState) SetTextMsgID(id string)      { s.textMsgID = id }
func (s *MessageState) SetReasoningID(id string)    { s.reasoningID = id }
func (s *MessageState) SetReasoningMsgID(id string) { s.reasoningMsgID = id }

// StartTextMessage 开始文本消息（幂等）。
// AG-UI 协议要求：文本消息必须以 TEXT_MESSAGE_START 开始，包含唯一的 messageId。
// 如果 textMsgID 为空，使用 ag-ui 提供的 ID 生成器生成 "msg-" 前缀的 UUID。
func (s *MessageState) StartTextMessage() []aguievents.Event {
	if s.textStarted {
		return nil
	}
	s.textStarted = true
	if s.textMsgID == "" {
		s.textMsgID = aguievents.GenerateMessageID()
	}
	return []aguievents.Event{aguievents.NewTextMessageStartEvent(s.textMsgID, aguievents.WithRole("assistant"))}
}

// EndTextMessage 结束文本消息（幂等）。
// AG-UI 协议要求：文本消息必须以 TEXT_MESSAGE_END 结束，messageId 必须与 START 匹配。
// 清除 textMsgID 以确保下次 StartTextMessage 生成新的 ID。
func (s *MessageState) EndTextMessage() []aguievents.Event {
	if !s.textStarted {
		return nil
	}
	s.textStarted = false
	id := s.textMsgID
	s.textMsgID = "" // 清除 ID，确保下次生成新 ID
	return []aguievents.Event{aguievents.NewTextMessageEndEvent(id)}
}

// StartReasoning 开始推理过程（幂等）。
// AG-UI 协议要求：推理过程以 REASONING_START 开始，关联到当前 textMsgID。
// 如果 reasoningID 为空，使用 ag-ui 提供的 ID 生成器生成 "msg-" 前缀的 UUID。
func (s *MessageState) StartReasoning() []aguievents.Event {
	if s.reasoningStarted {
		return nil
	}
	s.reasoningStarted = true
	if s.reasoningID == "" {
		if s.textMsgID != "" {
			s.reasoningID = s.textMsgID
		} else {
			s.reasoningID = aguievents.GenerateMessageID()
		}
	}
	return []aguievents.Event{aguievents.NewReasoningStartEvent(s.reasoningID)}
}

// EndReasoning 结束推理过程（幂等）。
// AG-UI 协议要求：推理过程以 REASONING_END 结束。
// 清除 reasoningID 以确保下次 StartReasoning 生成新的 ID。
func (s *MessageState) EndReasoning() []aguievents.Event {
	if !s.reasoningStarted {
		return nil
	}
	s.reasoningStarted = false
	id := s.reasoningID
	s.reasoningID = "" // 清除 ID，确保下次生成新 ID
	return []aguievents.Event{aguievents.NewReasoningEndEvent(id)}
}

// StartReasoningMessage 开始推理消息（幂等）。
// AG-UI 协议要求：推理消息以 REASONING_MESSAGE_START 开始，有独立的 messageId。
// 如果 reasoningMsgID 为空，使用 ag-ui 提供的 ID 生成器生成 "msg-" 前缀的 UUID。
func (s *MessageState) StartReasoningMessage() []aguievents.Event {
	if s.reasoningMsgStarted {
		return nil
	}
	s.reasoningMsgStarted = true
	if s.reasoningMsgID == "" {
		s.reasoningMsgID = aguievents.GenerateMessageID()
	}
	return []aguievents.Event{aguievents.NewReasoningMessageStartEvent(s.reasoningMsgID, "reasoning")}
}

// EndReasoningMessage 结束推理消息（幂等）。
// AG-UI 协议要求：推理消息以 REASONING_MESSAGE_END 结束。
// 清除 reasoningMsgID 以确保下次 StartReasoningMessage 生成新的 ID。
func (s *MessageState) EndReasoningMessage() []aguievents.Event {
	if !s.reasoningMsgStarted {
		return nil
	}
	s.reasoningMsgStarted = false
	id := s.reasoningMsgID
	s.reasoningMsgID = "" // 清除 ID，确保下次生成新 ID
	return []aguievents.Event{aguievents.NewReasoningMessageEndEvent(id)}
}

// EndAll 结束所有活跃消息（幂等）。
// 用于 Tool 消息处理和 Agent 切换等场景，确保所有未结束的消息被正确关闭。
// 调用顺序：REASONING_MESSAGE_END → REASONING_END → TEXT_MESSAGE_END
func (s *MessageState) EndAll() []aguievents.Event {
	var events []aguievents.Event
	events = append(events, s.EndReasoningMessage()...)
	events = append(events, s.EndReasoning()...)
	events = append(events, s.EndTextMessage()...)
	return events
}

// Reset 重置所有状态。
// 用于 ResetMessageID 场景，清除所有消息 ID 和状态标志。
func (s *MessageState) Reset() {
	s.textMsgID = ""
	s.reasoningID = ""
	s.reasoningMsgID = ""
	s.textStarted = false
	s.reasoningStarted = false
	s.reasoningMsgStarted = false
}
