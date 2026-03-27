package request

type APIKeyStatisticReq struct {
	APIKeyIds   []string `json:"apiKeyIds" `  // API Key 列表
	MethodPaths []string `json:"methodPaths"` // OpenAPI方法+路径（例如：POST-/agent/chat）
	StartDate   string   `json:"startDate" `  // 开始时间（格式yyyy-mm-dd）
	EndDate     string   `json:"endDate" `    // 结束时间（格式yyyy-mm-dd）
}

func (r *APIKeyStatisticReq) Check() error { return nil }

type APIKeyStatisticListReq struct {
	APIKeyIds   []string `json:"apiKeyIds" `   // API Key 列表
	MethodPaths []string `json:"methodPaths" ` // OpenAPI方法+路径（例如：POST-/agent/chat）
	StartDate   string   `json:"startDate" `   // 开始时间（格式yyyy-mm-dd）
	EndDate     string   `json:"endDate" `     // 结束时间（格式yyyy-mm-dd）
	PageNo      int      `json:"pageNo" `      // 页面编号，从1开始
	PageSize    int      `json:"pageSize"`     // 单页数量
}

func (r *APIKeyStatisticListReq) Check() error { return nil }

type APIKeyStatisticRecordReq struct {
	APIKeyIds   []string `json:"apiKeyIds" `   // API Key 列表
	MethodPaths []string `json:"methodPaths" ` // OpenAPI方法+路径（例如：POST-/agent/chat）
	StartDate   string   `json:"startDate" `   // 开始时间（格式yyyy-mm-dd）
	EndDate     string   `json:"endDate" `     // 结束时间（格式yyyy-mm-dd）
	PageNo      int      `json:"pageNo" `      // 页面编号，从1开始
	PageSize    int      `json:"pageSize" `    // 单页数量
}

func (r *APIKeyStatisticRecordReq) Check() error { return nil }

type ExportAPIKeyStatisticListReq struct {
	APIKeyIds   []string `json:"apiKeyIds" `   // API Key 列表
	MethodPaths []string `json:"methodPaths" ` // OpenAPI方法+路径（例如：POST-/agent/chat）
	StartDate   string   `json:"startDate" `   // 开始时间（格式yyyy-mm-dd）
	EndDate     string   `json:"endDate" `     // 结束时间（格式yyyy-mm-dd）
}

func (r *ExportAPIKeyStatisticListReq) Check() error { return nil }

type ExportAPIKeyStatisticRecordReq struct {
	APIKeyIds   []string `json:"apiKeyIds"`   // API Key 列表
	MethodPaths []string `json:"methodPaths"` // OpenAPI方法+路径（例如：POST-/agent/chat）
	StartDate   string   `json:"startDate" `  // 开始时间（格式yyyy-mm-dd）
	EndDate     string   `json:"endDate" `    // 结束时间（格式yyyy-mm-dd）
}

func (r *ExportAPIKeyStatisticRecordReq) Check() error { return nil }
