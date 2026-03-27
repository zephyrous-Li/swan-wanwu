package model

type ModelStatistic struct {
	ID        uint32 `gorm:"primary_key"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_model_statistic_created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`

	// 唯一键（用于每日维度聚合）
	OrgID string `gorm:"size:64;uniqueIndex:idx_model_statistic_unique,priority:1"`
	// 用户ID
	UserID string `gorm:"size:64;uniqueIndex:idx_model_statistic_unique,priority:2"`
	// 模型 ID
	ModelID string `gorm:"size:64;uniqueIndex:idx_model_statistic_unique,priority:3"`
	// 模型供应商
	Provider string `gorm:"size:64;uniqueIndex:idx_model_statistic_unique,priority:4"`
	// 日期
	Date string `gorm:"size:16;uniqueIndex:idx_model_statistic_unique,priority:5"`

	// 模型名称
	Model string `gorm:"size:128;index:idx_model_statistic_model"`
	// 模型类型（大语言、视觉、ocr、gui）
	ModelType string `gorm:"size:32;index:idx_model_statistic_model_type"`
	// prompt tokens
	PromptTokens int64 `gorm:"index:idx_model_statistic_prompt_tokens"`
	// completion tokens
	CompletionTokens int64 `gorm:"index:idx_model_statistic_completion_tokens"`
	// total tokens
	TotalTokens int64 `gorm:"index:idx_model_statistic_total_tokens"`
	// 首token时延 (流式ms)
	FirstTokenLatency int64 `gorm:"index:idx_model_statistic_first_token_latency"`
	// 耗时 (非流式ms)
	Costs int64 `gorm:"index:idx_model_statistic_costs"`
	// call count
	CallCount int32 `gorm:"index:idx_model_statistic_call_count"`
	// 流式调用次数
	StreamCount int32 `gorm:"index:idx_model_statistic_stream_count"`
	// 非流式调用次数
	NonStreamCount int32 `gorm:"index:idx_model_statistic_non_stream_count"`
	// call failure
	CallFailure int32 `gorm:"index:idx_model_statistic_call_failure"`
	// 流式调用次数失败
	StreamFailure int32 `gorm:"index:idx_model_statistic_stream_failure"`
	// 非流式调用次数失败
	NonStreamFailure int32 `gorm:"index:idx_model_statistic_non_stream_failure"`
}
