package ag_ui_util

import (
	"context"
	"encoding/json"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/google/uuid"
	"github.com/sst/opencode-sdk-go"

	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
)

type opencodeEventType string

const (
	opencodeEventTypeText      opencodeEventType = "text"
	opencodeEventTypeToolUse   opencodeEventType = "tool_use"
	opencodeEventTypeReasoning opencodeEventType = "reasoning"
)

type opencodeEvent struct {
	Type opencodeEventType `json:"type"`
	Part json.RawMessage   `json:"part"`
}

type opencodeErrorPart struct {
	Error struct {
		Name string `json:"name"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	} `json:"error"`
}

type OpencodeTranslator struct {
	BaseState
	activeToolCalls map[string]bool
}

func NewOpencodeTranslator(runID, threadID string) *OpencodeTranslator {
	return &OpencodeTranslator{
		BaseState:       NewBaseState(runID, threadID),
		activeToolCalls: make(map[string]bool),
	}
}

func (t *OpencodeTranslator) TranslateStream(ctx context.Context, in <-chan string) <-chan aguievents.Event {
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
			select {
			case <-ctx.Done():
				return
			case line, ok := <-in:
				if !ok {
					return
				}
				for _, evt := range t.translate(ctx, line) {
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

func (t *OpencodeTranslator) translate(_ context.Context, line string) []aguievents.Event {
	var evt opencodeEvent
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		log.Warnf("[ag-ui-util][%s] failed to parse opencode event: %v", t.RunID(), err)
		return nil
	}

	if t.MessageID() == "" {
		t.SetMessageID(uuid.NewString())
	}

	var events []aguievents.Event

	switch evt.Type {
	case opencodeEventTypeText:
		events = t.translateText(evt.Part)
	case opencodeEventTypeToolUse:
		events = t.translateToolUse(evt.Part)
	case opencodeEventTypeReasoning:
		events = t.translateReasoning(evt.Part)
	case "error":
		events = t.translateError(evt.Part)
	}

	return events
}

func (t *OpencodeTranslator) translateText(partData json.RawMessage) []aguievents.Event {
	var part opencode.TextPart
	if err := json.Unmarshal(partData, &part); err != nil {
		log.Warnf("[ag-ui-util][%s] failed to parse text part: %v", t.RunID(), err)
		return nil
	}

	if part.Text == "" {
		return nil
	}

	var events []aguievents.Event
	events = append(events, t.EnsureRunStarted()...)
	events = append(events, t.EndReasoningMessage()...)
	events = append(events, t.EndReasoning()...)
	events = append(events, t.StartTextMessage()...)
	events = append(events, aguievents.NewTextMessageContentEvent(t.MessageID(), part.Text))
	return events
}

func (t *OpencodeTranslator) translateReasoning(partData json.RawMessage) []aguievents.Event {
	var part opencode.ReasoningPart
	if err := json.Unmarshal(partData, &part); err != nil {
		log.Warnf("[ag-ui-util][%s] failed to parse reasoning part: %v", t.RunID(), err)
		return nil
	}

	if part.Text == "" {
		return nil
	}

	var events []aguievents.Event
	events = append(events, t.EnsureRunStarted()...)
	events = append(events, t.EndTextMessage()...)
	events = append(events, t.StartReasoning()...)
	events = append(events, t.StartReasoningMessage()...)
	events = append(events, aguievents.NewReasoningMessageContentEvent(t.ReasoningMessageID(), part.Text))
	return events
}

func (t *OpencodeTranslator) translateError(partData json.RawMessage) []aguievents.Event {
	var part opencodeErrorPart
	if err := json.Unmarshal(partData, &part); err != nil {
		log.Warnf("[ag-ui-util][%s] failed to parse error part: %v", t.RunID(), err)
		return nil
	}

	msg := "[error] " + part.Error.Name
	if part.Error.Data.Message != "" {
		msg += ": " + part.Error.Data.Message
	}

	var events []aguievents.Event
	events = append(events, t.EnsureRunStarted()...)
	events = append(events, t.EndReasoningMessage()...)
	events = append(events, t.EndReasoning()...)
	events = append(events, t.StartTextMessage()...)
	events = append(events, aguievents.NewTextMessageContentEvent(t.MessageID(), msg))
	return events
}

func (t *OpencodeTranslator) translateToolUse(partData json.RawMessage) []aguievents.Event {
	var part opencode.ToolPart
	if err := json.Unmarshal(partData, &part); err != nil {
		log.Warnf("[ag-ui-util][%s] failed to parse tool_use part: %v", t.RunID(), err)
		return nil
	}

	var events []aguievents.Event
	events = append(events, t.EnsureRunStarted()...)
	events = append(events, t.EndReasoningMessage()...)
	events = append(events, t.EndReasoning()...)

	if t.MessageID() == "" {
		t.SetMessageID(uuid.NewString())
	}

	toolCallID := part.CallID
	if toolCallID == "" {
		toolCallID = part.ID
	}

	switch part.State.Status {
	case opencode.ToolPartStateStatusPending, opencode.ToolPartStateStatusRunning:
		if !t.activeToolCalls[toolCallID] {
			events = append(events, aguievents.NewToolCallStartEvent(toolCallID, part.Tool, aguievents.WithParentMessageID(t.MessageID())))
			t.activeToolCalls[toolCallID] = true
			if input := t.getToolInput(part.State); input != "" {
				events = append(events, aguievents.NewToolCallArgsEvent(toolCallID, input))
			}
		}

	case opencode.ToolPartStateStatusCompleted:
		if !t.activeToolCalls[toolCallID] {
			events = append(events, aguievents.NewToolCallStartEvent(toolCallID, part.Tool, aguievents.WithParentMessageID(t.MessageID())))
			if input := t.getToolInput(part.State); input != "" {
				events = append(events, aguievents.NewToolCallArgsEvent(toolCallID, input))
			}
			events = append(events, aguievents.NewToolCallEndEvent(toolCallID))
		} else {
			events = append(events, aguievents.NewToolCallEndEvent(toolCallID))
			delete(t.activeToolCalls, toolCallID)
		}
		resultMessageID := uuid.NewString()
		events = append(events, aguievents.NewToolCallResultEvent(resultMessageID, toolCallID, part.State.Output))

	case opencode.ToolPartStateStatusError:
		if !t.activeToolCalls[toolCallID] {
			events = append(events, aguievents.NewToolCallStartEvent(toolCallID, part.Tool, aguievents.WithParentMessageID(t.MessageID())))
			events = append(events, aguievents.NewToolCallEndEvent(toolCallID))
		} else {
			events = append(events, aguievents.NewToolCallEndEvent(toolCallID))
			delete(t.activeToolCalls, toolCallID)
		}
		resultMessageID := uuid.NewString()
		events = append(events, aguievents.NewToolCallResultEvent(resultMessageID, toolCallID, part.State.Error))
	}

	return events
}

func (t *OpencodeTranslator) getToolInput(state opencode.ToolPartState) string {
	if state.Input == nil {
		return ""
	}
	switch v := state.Input.(type) {
	case string:
		return v
	case map[string]interface{}:
		data, _ := json.Marshal(v)
		return string(data)
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}
