package ag_ui_util

import (
	"context"
	"io"

	"github.com/UnicomAI/wanwu/pkg/util"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type AgentActivitySimple struct {
	activityType        string
	activityID          string
	agentName           string
	reasoningID         string
	reasoningMsgID      string
	textMsgID           string
	reasoningStarted    bool
	reasoningMsgStarted bool
	textStarted         bool
	toolCallStarted     map[string]bool
	toolCallMsgIDs      map[string]string
}

func NewAgentActivitySimple(agentName, activityType string) *AgentActivitySimple {
	return &AgentActivitySimple{
		activityType:    activityType,
		activityID:      uuid.NewString(),
		agentName:       agentName,
		toolCallStarted: make(map[string]bool),
		toolCallMsgIDs:  make(map[string]string),
	}
}

type EinoMultiAgentSimpleTranslator struct {
	runID              string
	threadID           string
	runStarted         bool
	runFinished        bool
	toolCallIDs        map[string]bool
	agentActivities    []*AgentActivitySimple
	currentActivity    *AgentActivitySimple
	agentInstanceCount map[string]int
}

func NewEinoMultiAgentSimpleTranslator(runID, threadID string) *EinoMultiAgentSimpleTranslator {
	return &EinoMultiAgentSimpleTranslator{
		runID:              runID,
		threadID:           threadID,
		toolCallIDs:        make(map[string]bool),
		agentActivities:    make([]*AgentActivitySimple, 0),
		agentInstanceCount: make(map[string]int),
	}
}

func (t *EinoMultiAgentSimpleTranslator) TranslateStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) <-chan aguievents.Event {
	out := make(chan aguievents.Event, 1024)
	go func() {
		defer util.PrintPanicStack()
		defer close(out)
		defer func() {
			for _, evt := range t.finishAllAgents() {
				select {
				case out <- evt:
				case <-ctx.Done():
					return
				}
			}
		}()

		select {
		case out <- aguievents.NewRunStartedEvent(t.threadID, t.runID):
			t.runStarted = true
		case <-ctx.Done():
			return
		}

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
				for _, evt := range t.translateMessageForCurrentAgent(errMsg) {
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

			agentName := event.AgentName
			if agentName == "" {
				agentName = "default"
			}

			shouldSwitch := t.currentActivity == nil || t.currentActivity.agentName != agentName
			if shouldSwitch {
				for _, evt := range t.switchAgent(agentName) {
					select {
					case out <- evt:
					case <-ctx.Done():
						return
					}
				}
			}

			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}

			msgOutput := event.Output.MessageOutput

			if msgOutput.IsStreaming {
				t.translateStreamForAgent(ctx, msgOutput, out)
			} else {
				for _, evt := range t.translateMessageForCurrentAgent(msgOutput.Message) {
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

func (t *EinoMultiAgentSimpleTranslator) switchAgent(newAgent string) []aguievents.Event {
	var events []aguievents.Event

	if t.currentActivity != nil {
		events = append(events, t.endCurrentAgentActivity()...)
	}

	activityType := t.getActivityType(newAgent)

	t.agentInstanceCount[newAgent]++
	instanceNum := t.agentInstanceCount[newAgent]

	activity := NewAgentActivitySimple(newAgent, activityType)
	t.agentActivities = append(t.agentActivities, activity)

	content := map[string]interface{}{
		"agentName":   newAgent,
		"instanceNum": instanceNum,
		"status":      "start",
	}
	events = append(events, aguievents.NewActivitySnapshotEvent(
		activity.activityID,
		activityType,
		content,
	))

	t.currentActivity = activity

	return events
}

func (t *EinoMultiAgentSimpleTranslator) getActivityType(agentName string) string {
	switch agentName {
	case "Plan Agent", "PlanAgent":
		return "plan_activity"
	case "Research Agent", "ResearchAgent":
		return "research_activity"
	case "Report Agent", "ReportAgent":
		return "report_activity"
	case "Supervisor Agent", "SupervisorAgent":
		return "supervisor_activity"
	case "Multi-Modal Agent", "MultiModalAgent", "Multi-ModalAgent":
		return "multimodal_activity"
	default:
		return "agent_activity"
	}
}

func (t *EinoMultiAgentSimpleTranslator) endCurrentAgentActivity() []aguievents.Event {
	if t.currentActivity == nil {
		return nil
	}

	activity := t.currentActivity
	var events []aguievents.Event

	if activity.reasoningMsgStarted {
		events = append(events, aguievents.NewReasoningMessageEndEvent(activity.reasoningMsgID))
		activity.reasoningMsgStarted = false
	}

	if activity.reasoningStarted {
		events = append(events, aguievents.NewReasoningEndEvent(activity.reasoningID))
		activity.reasoningStarted = false
	}

	if activity.textStarted {
		events = append(events, aguievents.NewTextMessageEndEvent(activity.textMsgID))
		activity.textStarted = false
	}

	content := map[string]interface{}{
		"agentName": activity.agentName,
		"status":    "finished",
	}
	events = append(events, aguievents.NewActivitySnapshotEvent(
		activity.activityID,
		activity.activityType,
		content,
	))

	return events
}

func (t *EinoMultiAgentSimpleTranslator) translateMessageForCurrentAgent(msg *schema.Message) []aguievents.Event {
	if t.currentActivity == nil {
		return nil
	}

	return t.translateMessageWithActivity(msg, t.currentActivity)
}

func (t *EinoMultiAgentSimpleTranslator) translateMessageWithActivity(msg *schema.Message, activity *AgentActivitySimple) []aguievents.Event {
	if msg == nil {
		return nil
	}

	var events []aguievents.Event

	if msg.Role == schema.Tool && msg.ToolCallID != "" {
		toolResultMessageID := uuid.NewString()
		events = append(events, aguievents.NewToolCallResultEvent(toolResultMessageID, msg.ToolCallID, msg.Content))
		return events
	}

	hasContent := msg.Content != "" || msg.ReasoningContent != "" || len(msg.ToolCalls) > 0
	if !hasContent {
		return nil
	}

	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			if tc.ID == "" || tc.Function.Name == "" {
				continue
			}
			if !t.toolCallIDs[tc.ID] {
				var toolCallMsgID string
				if activity.toolCallMsgIDs[tc.ID] == "" {
					toolCallMsgID = uuid.NewString()
					activity.toolCallMsgIDs[tc.ID] = toolCallMsgID
				} else {
					toolCallMsgID = activity.toolCallMsgIDs[tc.ID]
				}
				if !activity.toolCallStarted[tc.ID] {
					events = append(events, aguievents.NewToolCallStartEvent(toolCallMsgID, tc.Function.Name))
					activity.toolCallStarted[tc.ID] = true
				}
				if tc.Function.Arguments != "" {
					events = append(events, aguievents.NewToolCallArgsEvent(toolCallMsgID, tc.Function.Arguments))
				}
				events = append(events, aguievents.NewToolCallEndEvent(toolCallMsgID))
				t.toolCallIDs[tc.ID] = true
			}
		}
	}

	if msg.ReasoningContent != "" {
		if activity.textStarted {
			events = append(events, aguievents.NewTextMessageEndEvent(activity.textMsgID))
			activity.textStarted = false
		}
		if !activity.reasoningStarted {
			activity.reasoningID = uuid.NewString()
			events = append(events, aguievents.NewReasoningStartEvent(activity.reasoningID))
			activity.reasoningStarted = true
		}
		if !activity.reasoningMsgStarted {
			activity.reasoningMsgID = uuid.NewString()
			events = append(events, aguievents.NewReasoningMessageStartEvent(activity.reasoningMsgID, "reasoning"))
			activity.reasoningMsgStarted = true
		}
		events = append(events, aguievents.NewReasoningMessageContentEvent(activity.reasoningMsgID, msg.ReasoningContent))
	}

	if msg.Content != "" {
		if activity.reasoningMsgStarted {
			events = append(events, aguievents.NewReasoningMessageEndEvent(activity.reasoningMsgID))
			activity.reasoningMsgStarted = false
		}
		if activity.reasoningStarted {
			events = append(events, aguievents.NewReasoningEndEvent(activity.reasoningID))
			activity.reasoningStarted = false
		}
		if !activity.textStarted {
			activity.textMsgID = uuid.NewString()
			events = append(events, aguievents.NewTextMessageStartEvent(activity.textMsgID, aguievents.WithRole("assistant")))
			activity.textStarted = true
		}
		events = append(events, aguievents.NewTextMessageContentEvent(activity.textMsgID, msg.Content))
	}

	return events
}

func (t *EinoMultiAgentSimpleTranslator) translateStreamForAgent(ctx context.Context, msgOutput *adk.MessageVariant, out chan<- aguievents.Event) {
	if msgOutput.MessageStream == nil {
		return
	}
	defer msgOutput.MessageStream.Close()

	if t.currentActivity == nil {
		return
	}

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

		for _, evt := range t.translateMessageWithActivity(frame, t.currentActivity) {
			select {
			case out <- evt:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (t *EinoMultiAgentSimpleTranslator) finishAllAgents() []aguievents.Event {
	var events []aguievents.Event

	if t.currentActivity != nil {
		events = append(events, t.endCurrentAgentActivity()...)
	}

	if !t.runFinished {
		t.runFinished = true
		events = append(events, aguievents.NewRunFinishedEvent(t.threadID, t.runID))
	}

	return events
}
