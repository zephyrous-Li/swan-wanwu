package request

type AppRecordRequest struct {
	UserID string `json:"userId"`

	OrgID string `json:"orgId"`

	AppID string `json:"appId"`

	AppType string `json:"appType"`

	IsSuccess bool `json:"isSuccess"`

	IsStream bool `json:"isStream"`

	StreamCosts int64 `json:"streamCosts"`

	NonStreamCosts int64 `json:"nonStreamCosts"`

	Source string `json:"source"`
}

func (a *AppRecordRequest) Check() error { return nil }
