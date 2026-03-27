package model

// APIKeyRecord API Key调用记录明细表（存储每次调用的详细信息）
type APIKeyRecord struct {
	ID        uint32 `gorm:"primary_key"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_api_key_record_created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`
	// 组织ID
	OrgID string `gorm:"index:idx_api_key_record_org_id"`
	// 用户ID
	UserID string `gorm:"index:idx_api_key_record_user_id"`
	// API Key ID
	APIKeyID string `gorm:"index:idx_api_key_record_api_key_id"`
	// 请求方法+路径 (如 POST-/agent/chat)
	MethodPath string `gorm:"index:idx_api_key_record_method_path"`
	// 调用时间
	CallTime int64 `gorm:"index:idx_api_key_record_call_time"`
	// 响应状态 (success/failure)
	ResponseStatus string `gorm:"index:idx_api_key_record_response_status"`
	// 是否流式调用
	IsStream bool `gorm:"index:idx_api_key_record_is_stream"`
	// 流式耗时 (ms)
	StreamCosts int64 `gorm:"index:idx_api_key_record_stream_costs"`
	// 非流式耗时 (ms)
	NonStreamCosts int64 `gorm:"index:idx_api_key_record_non_stream_costs"`
	// 请求体 (JSON)
	RequestBody string `gorm:"type:text"`
	// 响应体 (JSON，流式调用为空)
	ResponseBody string `gorm:"type:text"`
	// 日期 (yyyy-mm-dd)
	Date string `gorm:"index:idx_api_key_record_date"`
}
