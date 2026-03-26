package v1

import (
	"net/http"

	v1 "github.com/UnicomAI/wanwu/internal/bff-service/server/http/handler/v1"
	mid "github.com/UnicomAI/wanwu/pkg/gin-util/mid-wrap"
	"github.com/gin-gonic/gin"
)

func registerAppStatistic(apiV1 *gin.RouterGroup) {
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app", http.MethodGet, v1.GetAppStatistic, "获取应用统计数据")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app/list", http.MethodGet, v1.GetAppStatisticList, "获取应用统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app/export", http.MethodGet, v1.ExportAppStatisticList, "导出应用统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app/select", http.MethodGet, v1.GetAppListSelect, "获取应用下拉列表")
}
