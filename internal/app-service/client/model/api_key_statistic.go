package model

// APIKeyStatistic API Key统计汇总表（按日期汇总）
type APIKeyStatistic struct {
	ID        uint32 `gorm:"primary_key"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_api_key_statistic_created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`

	// 唯一键（用于每日维度聚合）
	OrgID string `gorm:"size:64;uniqueIndex:idx_api_key_statistic_unique,priority:1"`
	// 用户ID
	UserID string `gorm:"size:64;uniqueIndex:idx_api_key_statistic_unique,priority:2"`
	// API Key ID
	APIKeyID string `gorm:"size:64;uniqueIndex:idx_api_key_statistic_unique,priority:3"`
	// 请求方法+路径 (如 POST-/agent/chat)
	MethodPath string `gorm:"size:128;uniqueIndex:idx_api_key_statistic_unique,priority:4"`
	// 日期 (yyyy-mm-dd)
	Date string `gorm:"size:16;uniqueIndex:idx_api_key_statistic_unique,priority:5"`

	// 调用次数
	CallCount int32 `gorm:"index:idx_api_key_statistic_call_count"`
	// 调用失败次数
	CallFailure int32 `gorm:"index:idx_api_key_statistic_call_failure"`
	// 流式调用次数
	StreamCount int32 `gorm:"index:idx_api_key_statistic_stream_count"`
	// 非流式调用次数
	NonStreamCount int32 `gorm:"index:idx_api_key_statistic_non_stream_count"`
	// 流式调用失败次数
	StreamFailure int32 `gorm:"index:idx_api_key_statistic_stream_failure"`
	// 非流式调用失败次数
	NonStreamFailure int32 `gorm:"index:idx_api_key_statistic_non_stream_failure"`
	// 流式耗时总和 (ms)
	StreamCosts int64 `gorm:"index:idx_api_key_statistic_stream_costs"`
	// 非流式耗时总和 (ms)
	NonStreamCosts int64 `gorm:"index:idx_api_key_statistic_non_stream_costs"`
}
