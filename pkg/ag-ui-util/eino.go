package ag_ui_util

import (
	"context"
	"fmt"
	"io"

	"github.com/UnicomAI/wanwu/pkg/util"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type EinoTranslator struct {
	BaseState
	toolCallIDs map[string]bool
}

func NewEinoTranslator(runID, threadID string) *EinoTranslator {
	return &EinoTranslator{
		BaseState:   NewBaseState(runID, threadID),
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
				t.SetMessageID(uuid.NewString())
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

// translateMessage 转换消息（支持非流式和流式帧）。
func (t *EinoTranslator) translateMessage(msg *schema.Message) []aguievents.Event {
	if msg == nil {
		return nil
	}

	// 处理工具调用结果
	if msg.Role == schema.Tool && msg.ToolCallID != "" {
		var events []aguievents.Event
		events = append(events, t.EnsureRunStarted()...)
		events = append(events, t.EndReasoningMessage()...)
		events = append(events, t.EndReasoning()...)
		events = append(events, t.EndTextMessage()...)
		events = append(events, aguievents.NewToolCallResultEvent(uuid.NewString(), msg.ToolCallID, msg.Content))
		return events
	}

	hasContent := msg.Content != "" || msg.ReasoningContent != "" || len(msg.ToolCalls) > 0
	if !hasContent {
		return nil
	}

	var events []aguievents.Event
	events = append(events, t.EnsureRunStarted()...)

	if len(msg.ToolCalls) > 0 {
		events = append(events, t.EndReasoningMessage()...)
		events = append(events, t.EndReasoning()...)
		events = append(events, t.EndTextMessage()...)

		for _, tc := range msg.ToolCalls {
			if tc.ID == "" || tc.Function.Name == "" {
				continue
			}
			fmt.Printf("tool call arg: %v\n", tc.Function.Arguments)
			if !t.toolCallIDs[tc.ID] {
				events = append(events, aguievents.NewToolCallStartEvent(tc.ID, tc.Function.Name, aguievents.WithParentMessageID(t.MessageID())))
				fmt.Printf("tool call start: %v\n", tc.Function)
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
