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

type AgentActivity struct {
	messageID           string
	activityType        string
	textStarted         bool
	reasoningStarted    bool
	reasoningMsgStarted bool
	toolCallStarted     map[string]bool
}

func NewAgentActivity(activityType string) *AgentActivity {
	return &AgentActivity{
		messageID:       uuid.NewString(),
		activityType:    activityType,
		toolCallStarted: make(map[string]bool),
	}
}

type EinoMultiAgentTranslator struct {
	runID           string
	threadID        string
	runStarted      bool
	runFinished     bool
	toolCallIDs     map[string]bool
	agentActivities map[string]*AgentActivity
	currentAgent    string
}

func NewEinoMultiAgentTranslator(runID, threadID string) *EinoMultiAgentTranslator {
	return &EinoMultiAgentTranslator{
		runID:           runID,
		threadID:        threadID,
		toolCallIDs:     make(map[string]bool),
		agentActivities: make(map[string]*AgentActivity),
	}
}

func (t *EinoMultiAgentTranslator) TranslateStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) <-chan aguievents.Event {
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

			if agentName != t.currentAgent {
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
				t.translateStreamForAgent(ctx, msgOutput, out, agentName)
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

func (t *EinoMultiAgentTranslator) switchAgent(newAgent string) []aguievents.Event {
	var events []aguievents.Event

	if t.currentAgent != "" && t.currentAgent != newAgent {
		events = append(events, t.endCurrentAgentActivity()...)
	}

	activityType := t.getActivityType(newAgent)

	if _, ok := t.agentActivities[newAgent]; !ok {
		activity := NewAgentActivity(activityType)
		t.agentActivities[newAgent] = activity
	}

	t.currentAgent = newAgent

	return events
}

func (t *EinoMultiAgentTranslator) getActivityType(agentName string) string {
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

func (t *EinoMultiAgentTranslator) endCurrentAgentActivity() []aguievents.Event {
	if t.currentAgent == "" {
		return nil
	}

	activity, ok := t.agentActivities[t.currentAgent]
	if !ok {
		return nil
	}

	var events []aguievents.Event

	if activity.reasoningMsgStarted {
		events = append(events, aguievents.NewActivityDeltaEvent(
			activity.messageID,
			activity.activityType,
			[]aguievents.JSONPatchOperation{
				{Value: aguievents.NewReasoningMessageEndEvent(activity.messageID)},
			},
		))
		activity.reasoningMsgStarted = false
	}

	if activity.reasoningStarted {
		events = append(events, aguievents.NewActivityDeltaEvent(
			activity.messageID,
			activity.activityType,
			[]aguievents.JSONPatchOperation{
				{Value: aguievents.NewReasoningEndEvent(activity.messageID)},
			},
		))
		activity.reasoningStarted = false
	}

	if activity.textStarted {
		events = append(events, aguievents.NewActivityDeltaEvent(
			activity.messageID,
			activity.activityType,
			[]aguievents.JSONPatchOperation{
				{Value: aguievents.NewTextMessageEndEvent(activity.messageID)},
			},
		))
		activity.textStarted = false
	}

	return events
}

func (t *EinoMultiAgentTranslator) translateMessageForCurrentAgent(msg *schema.Message) []aguievents.Event {
	if t.currentAgent == "" {
		return nil
	}

	activity, ok := t.agentActivities[t.currentAgent]
	if !ok {
		return nil
	}

	return t.translateMessageWithActivity(msg, activity)
}

func (t *EinoMultiAgentTranslator) translateMessageWithActivity(msg *schema.Message, activity *AgentActivity) []aguievents.Event {
	if msg == nil {
		return nil
	}

	if msg.Role == schema.Tool && msg.ToolCallID != "" {
		return []aguievents.Event{
			aguievents.NewActivityDeltaEvent(
				activity.messageID,
				activity.activityType,
				[]aguievents.JSONPatchOperation{
					{Value: aguievents.NewToolCallResultEvent(activity.messageID, msg.ToolCallID, msg.Content)},
				},
			),
		}
	}

	hasContent := msg.Content != "" || msg.ReasoningContent != "" || len(msg.ToolCalls) > 0
	if !hasContent {
		return nil
	}

	var events []aguievents.Event

	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			if tc.ID == "" || tc.Function.Name == "" {
				continue
			}
			if !t.toolCallIDs[tc.ID] {
				if !activity.toolCallStarted[tc.ID] {
					events = append(events, aguievents.NewActivityDeltaEvent(
						activity.messageID,
						activity.activityType,
						[]aguievents.JSONPatchOperation{
							{Value: aguievents.NewToolCallStartEvent(tc.ID, tc.Function.Name, aguievents.WithParentMessageID(activity.messageID))},
						},
					))
					activity.toolCallStarted[tc.ID] = true
				}
				if tc.Function.Arguments != "" {
					events = append(events, aguievents.NewActivityDeltaEvent(
						activity.messageID,
						activity.activityType,
						[]aguievents.JSONPatchOperation{
							{Value: aguievents.NewToolCallArgsEvent(tc.ID, tc.Function.Arguments)},
						},
					))
				}
				events = append(events, aguievents.NewActivityDeltaEvent(
					activity.messageID,
					activity.activityType,
					[]aguievents.JSONPatchOperation{
						{Value: aguievents.NewToolCallEndEvent(tc.ID)},
					},
				))
				t.toolCallIDs[tc.ID] = true
			}
		}
	}

	if msg.ReasoningContent != "" {
		if activity.textStarted {
			events = append(events, aguievents.NewActivityDeltaEvent(
				activity.messageID,
				activity.activityType,
				[]aguievents.JSONPatchOperation{
					{Value: aguievents.NewTextMessageEndEvent(activity.messageID)},
				},
			))
			activity.textStarted = false
		}
		if !activity.reasoningStarted {
			events = append(events, aguievents.NewActivityDeltaEvent(
				activity.messageID,
				activity.activityType,
				[]aguievents.JSONPatchOperation{
					{Value: aguievents.NewReasoningStartEvent(activity.messageID)},
				},
			))
			activity.reasoningStarted = true
		}
		if !activity.reasoningMsgStarted {
			events = append(events, aguievents.NewActivityDeltaEvent(
				activity.messageID,
				activity.activityType,
				[]aguievents.JSONPatchOperation{
					{Value: aguievents.NewReasoningMessageStartEvent(activity.messageID, "reasoning")},
				},
			))
			activity.reasoningMsgStarted = true
		}
		events = append(events, aguievents.NewActivityDeltaEvent(
			activity.messageID,
			activity.activityType,
			[]aguievents.JSONPatchOperation{
				{Value: aguievents.NewReasoningMessageContentEvent(activity.messageID, msg.ReasoningContent)},
			},
		))
	}

	if msg.Content != "" {
		if activity.reasoningMsgStarted {
			events = append(events, aguievents.NewActivityDeltaEvent(
				activity.messageID,
				activity.activityType,
				[]aguievents.JSONPatchOperation{
					{Value: aguievents.NewReasoningMessageEndEvent(activity.messageID)},
				},
			))
			activity.reasoningMsgStarted = false
		}
		if activity.reasoningStarted {
			events = append(events, aguievents.NewActivityDeltaEvent(
				activity.messageID,
				activity.activityType,
				[]aguievents.JSONPatchOperation{
					{Value: aguievents.NewReasoningEndEvent(activity.messageID)},
				},
			))
			activity.reasoningStarted = false
		}
		if !activity.textStarted {
			events = append(events, aguievents.NewActivityDeltaEvent(
				activity.messageID,
				activity.activityType,
				[]aguievents.JSONPatchOperation{
					{Value: aguievents.NewTextMessageStartEvent(activity.messageID, aguievents.WithRole("assistant"))},
				},
			))
			activity.textStarted = true
		}
		events = append(events, aguievents.NewActivityDeltaEvent(
			activity.messageID,
			activity.activityType,
			[]aguievents.JSONPatchOperation{
				{Value: aguievents.NewTextMessageContentEvent(activity.messageID, msg.Content)},
			},
		))
	}

	return events
}

func (t *EinoMultiAgentTranslator) translateStreamForAgent(ctx context.Context, msgOutput *adk.MessageVariant, out chan<- aguievents.Event, agentID string) {
	if msgOutput.MessageStream == nil {
		return
	}
	defer msgOutput.MessageStream.Close()

	activity, ok := t.agentActivities[agentID]
	if !ok {
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

		for _, evt := range t.translateMessageWithActivity(frame, activity) {
			select {
			case out <- evt:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (t *EinoMultiAgentTranslator) finishAllAgents() []aguievents.Event {
	var events []aguievents.Event

	if t.currentAgent != "" {
		events = append(events, t.endCurrentAgentActivity()...)
	}

	if !t.runFinished {
		t.runFinished = true
		events = append(events, aguievents.NewRunFinishedEvent(t.threadID, t.runID))
	}

	return events
}
