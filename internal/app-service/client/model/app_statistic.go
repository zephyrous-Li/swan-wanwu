package model

type AppStatistic struct {
	ID        uint32 `gorm:"primary_key"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_app_statistic_created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli"`

	// 唯一键（用于每日维度聚合）
	OrgID   string `gorm:"size:64;uniqueIndex:idx_app_statistic_unique,priority:1"`
	UserID  string `gorm:"size:64;uniqueIndex:idx_app_statistic_unique,priority:2"`
	AppID   string `gorm:"size:64;uniqueIndex:idx_app_statistic_unique,priority:3"`
	AppType string `gorm:"size:64;uniqueIndex:idx_app_statistic_unique,priority:4"` // agent/rag/workflow/chatflow
	Date    string `gorm:"size:16;uniqueIndex:idx_app_statistic_unique,priority:5"` // yyyy-mm-dd

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
}
