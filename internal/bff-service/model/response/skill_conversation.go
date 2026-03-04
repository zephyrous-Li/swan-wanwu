package response

type CreateSkillConversationResp struct {
	ConversationId string `json:"conversationId"` // 会话ID
}

type SkillConversationItem struct {
	ConversationId string `json:"conversationId"` // 会话ID
	Title          string `json:"title"`          // 会话标题
	CreatedAt      string `json:"createdAt"`      // 创建时间
}

type SkillConversationDetailInfo struct {
	ConversationDetailInfo
	ResponseFiles []AssistantResponseFile `json:"responseFiles"`
}

type AssistantResponseFile struct {
	FileName string `json:"name"`
	FileSize int64  `json:"size"`
	FileUrl  string `json:"fileUrl"`
	MIMEType string `json:"mimeType"`

	// 扩展信息：
	// skill => {"name":"技能名称", "desc":"技能描述", "author":"技能作者", "avatar":{"path":"技能图标"}, "inResource": bool, "expiredAt": "过期时间7天", "skillSaveId": "保存的技能ID"}
	MetaData map[string]interface{} `json:"metadata"`
}
