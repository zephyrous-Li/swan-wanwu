package request

type OpenAPIAgentCreateConversationRequest struct {
	Title string `json:"title"`
	UUID  string `json:"uuid" validate:"required"`
}

func (req *OpenAPIAgentCreateConversationRequest) Check() error {
	return nil
}

type OpenAPIAgentChatRequest struct {
	UUID           string                 `json:"uuid" validate:"required"`
	ConversationID string                 `json:"conversation_id" validate:"required"`
	Query          string                 `json:"query" validate:"required"`
	Stream         bool                   `json:"stream"`
	FileInfo       []ConversionStreamFile `json:"file_info"`
}

func (req *OpenAPIAgentChatRequest) Check() error {
	return nil
}

type OpenAPIRagChatRequest struct {
	UUID    string     `json:"uuid" validate:"required"`
	Query   string     `json:"query" validate:"required"`
	Stream  bool       `json:"stream"`
	History []*History `json:"history"`
}

func (req *OpenAPIRagChatRequest) Check() error {
	return nil
}

type OpenAPIWorkflowRunReq struct {
	UUID       string         `json:"uuid" validate:"required"`
	Parameters map[string]any `json:"parameters"`
}

func (req *OpenAPIWorkflowRunReq) Check() error {
	return nil
}

type OpenAPIChatflowCreateConversationRequest struct {
	UUID             string `json:"uuid" validate:"required"`
	ConversationName string `json:"conversation_name"`
}

func (req *OpenAPIChatflowCreateConversationRequest) Check() error {
	return nil
}

type OpenAPIChatflowChatRequest struct {
	UUID           string         `json:"uuid" validate:"required"`
	ConversationId string         `json:"conversation_id" validate:"required"`
	Query          string         `json:"query" validate:"required"`
	Parameters     map[string]any `json:"parameters"`
}

func (req *OpenAPIChatflowChatRequest) Check() error {
	return nil
}

type OpenAPIChatflowGetConversationMessageListRequest struct {
	UUID           string `json:"uuid" validate:"required"`
	ConversationId string `json:"conversation_id" validate:"required"`
	Limit          string `json:"limit"`
}

func (req *OpenAPIChatflowGetConversationMessageListRequest) Check() error {
	return nil
}
