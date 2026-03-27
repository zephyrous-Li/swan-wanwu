package v1

import (
	"net/http"

	v1 "github.com/UnicomAI/wanwu/internal/bff-service/server/http/handler/v1"
	mid "github.com/UnicomAI/wanwu/pkg/gin-util/mid-wrap"
	"github.com/gin-gonic/gin"
)

func registerAPIKeyStatistic(apiV1 *gin.RouterGroup) {
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api", http.MethodPost, v1.GetAPIKeyStatistic, "获取API Key统计数据")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/list", http.MethodPost, v1.GetAPIKeyStatisticList, "获取API Key调用统计")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/record", http.MethodPost, v1.GetAPIKeyStatisticRecord, "获取API Key调用记录")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/list/export", http.MethodPost, v1.ExportAPIKeyStatisticList, "导出API Key统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/record/export", http.MethodPost, v1.ExportAPIKeyStatisticRecord, "导出API Key调用记录")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/routes", http.MethodGet, v1.GetApiKeyStatisticRoutes, "获取API Key统计路由列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/select", http.MethodGet, v1.GetAPIKeySelect, "获取API Key统计路由列表")
}
