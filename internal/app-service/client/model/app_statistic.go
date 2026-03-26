package model

type AppRecord struct {
	ID        uint32 `gorm:"primary_key"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_app_record_created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`

	// 基础信息
	OrgID   string `gorm:"index:idx_app_record_org_id"`
	UserID  string `gorm:"index:idx_app_record_user_id"`
	AppID   string `gorm:"index:idx_app_record_app_id"`
	AppType string `gorm:"index:idx_app_record_app_type"` // agent/rag/workflow/chatflow

	// 调用统计
	CallCount   int32
	CallFailure int32

	// 流式统计
	StreamCount   int32
	StreamFailure int32
	StreamCosts   int64 // 流式耗时总和

	// 非流式统计
	NonStreamCount   int32
	NonStreamFailure int32
	NonStreamCosts   int64 // 非流式耗时总和

	// 渠道统计
	WebCallCount       int32
	WebCallFailure     int32
	OpenapiCallCount   int32
	OpenapiCallFailure int32

	// 智能体专属
	WebUrlCallCount   int32
	WebUrlCallFailure int32

	// 日期
	Date string `gorm:"index:idx_app_record_date"`
}
