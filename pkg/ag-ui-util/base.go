package ag_ui_util

import (
	"context"
	"encoding/json"

	"github.com/UnicomAI/wanwu/pkg/util"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/google/uuid"
)

type BaseState struct {
	runID               string
	threadID            string
	messageID           string
	reasoningMessageID  string
	runStarted          bool
	runFinished         bool
	textStarted         bool
	reasoningStarted    bool
	reasoningMsgStarted bool
}

func NewBaseState(runID, threadID string) BaseState {
	return BaseState{
		runID:    runID,
		threadID: threadID,
	}
}

func (s *BaseState) RunID() string              { return s.runID }
func (s *BaseState) ThreadID() string           { return s.threadID }
func (s *BaseState) MessageID() string          { return s.messageID }
func (s *BaseState) ReasoningMessageID() string { return s.reasoningMessageID }

func (s *BaseState) SetMessageID(messageID string) {
	s.messageID = messageID
}

func (s *BaseState) EnsureRunStarted() []aguievents.Event {
	if s.runStarted {
		return nil
	}
	s.runStarted = true
	return []aguievents.Event{aguievents.NewRunStartedEvent(s.threadID, s.runID)}
}

func (s *BaseState) StartTextMessage() []aguievents.Event {
	if s.textStarted {
		return nil
	}
	s.textStarted = true
	if s.messageID == "" {
		s.messageID = uuid.NewString()
	}
	return []aguievents.Event{aguievents.NewTextMessageStartEvent(s.messageID, aguievents.WithRole("assistant"))}
}

func (s *BaseState) EndTextMessage() []aguievents.Event {
	if !s.textStarted {
		return nil
	}
	s.textStarted = false
	return []aguievents.Event{aguievents.NewTextMessageEndEvent(s.messageID)}
}

func (s *BaseState) StartReasoning() []aguievents.Event {
	if s.reasoningStarted {
		return nil
	}
	s.reasoningStarted = true
	return []aguievents.Event{aguievents.NewReasoningStartEvent(s.messageID)}
}

func (s *BaseState) StartReasoningMessage() []aguievents.Event {
	if s.reasoningMsgStarted {
		return nil
	}
	s.reasoningMsgStarted = true
	if s.reasoningMessageID == "" {
		s.reasoningMessageID = uuid.NewString()
	}
	return []aguievents.Event{aguievents.NewReasoningMessageStartEvent(s.reasoningMessageID, "reasoning")}
}

func (s *BaseState) EndReasoningMessage() []aguievents.Event {
	if !s.reasoningMsgStarted {
		return nil
	}
	s.reasoningMsgStarted = false
	return []aguievents.Event{aguievents.NewReasoningMessageEndEvent(s.reasoningMessageID)}
}

func (s *BaseState) EndReasoning() []aguievents.Event {
	if !s.reasoningStarted {
		return nil
	}
	s.reasoningStarted = false
	return []aguievents.Event{aguievents.NewReasoningEndEvent(s.messageID)}
}

func (s *BaseState) FinishBase() []aguievents.Event {
	if s.runFinished {
		return nil
	}
	s.runFinished = true
	var events []aguievents.Event
	events = append(events, s.EnsureRunStarted()...)
	events = append(events, s.EndReasoningMessage()...)
	events = append(events, s.EndReasoning()...)
	events = append(events, s.EndTextMessage()...)
	events = append(events, aguievents.NewRunFinishedEvent(s.threadID, s.runID))
	return events
}

func EventsToJSONChannel(ctx context.Context, in <-chan aguievents.Event) <-chan string {
	out := make(chan string, 1024)
	go func() {
		defer util.PrintPanicStack()
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-in:
				if !ok {
					return
				}
				if data, err := json.Marshal(evt); err == nil {
					out <- string(data)
				}
			}
		}
	}()
	return out
}

func RemoveReasoningContent(content string) string {
	return content
}
