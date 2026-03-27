package model

// APIKeyStatistic API Key统计汇总表（按日期汇总）
type APIKeyStatistic struct {
	ID        uint32 `gorm:"primary_key"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_api_key_stat_created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`
	// 组织ID
	OrgID string `gorm:"index:idx_api_key_stat_org_id"`
	// 用户ID
	UserID string `gorm:"index:idx_api_key_stat_user_id"`
	// API Key ID
	APIKeyID string `gorm:"index:idx_api_key_stat_api_key_id"`
	// 请求方法+路径 (如 POST-/agent/chat)
	MethodPath string `gorm:"index:idx_api_key_stat_method_path"`
	// 调用次数
	CallCount int32 `gorm:"index:idx_api_key_stat_call_count"`
	// 调用失败次数
	CallFailure int32 `gorm:"index:idx_api_key_stat_call_failure"`
	// 流式调用次数
	StreamCount int32 `gorm:"index:idx_api_key_stat_stream_count"`
	// 非流式调用次数
	NonStreamCount int32 `gorm:"index:idx_api_key_stat_non_stream_count"`
	// 流式调用失败次数
	StreamFailure int32 `gorm:"index:idx_api_key_stat_stream_failure"`
	// 非流式调用失败次数
	NonStreamFailure int32 `gorm:"index:idx_api_key_stat_non_stream_failure"`
	// 流式耗时总和 (ms)
	StreamCosts int64 `gorm:"index:idx_api_key_stat_stream_costs"`
	// 非流式耗时总和 (ms)
	NonStreamCosts int64 `gorm:"index:idx_api_key_stat_non_stream_costs"`
	// 日期 (yyyy-mm-dd)
	Date string `gorm:"index:idx_api_key_stat_date"`
}
