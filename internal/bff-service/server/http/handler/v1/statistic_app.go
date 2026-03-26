package v1

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetAppStatistic
//
//	@Tags			app_observability.statistic
//	@Summary		获取应用统计数据
//	@Description	获取应用统计数据
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			startDate	query		string	true	"开始时间（格式yyyy-mm-dd）"
//	@Param			endDate		query		string	true	"结束时间（格式yyyy-mm-dd）"
//	@Param			apps		query		string	false	"应用ID列表"
//	@Param			appType		query		string	false	"应用类型（默认agent）"
//	@Success		200			{object}	response.Response{data=response.AppStatistic}
//	@Router			/statistic/app [get]
func GetAppStatistic(ctx *gin.Context) {
	startDate := ctx.Query("startDate")
	endDate := ctx.Query("endDate")
	appIds := ctx.Query("apps")
	appType := ctx.Query("appType")
	var appIdList []string
	if appIds != "" {
		appIdList = strings.Split(appIds, ",")
	}
	resp, err := service.GetAppStatistic(ctx, getUserID(ctx), getOrgID(ctx), startDate, endDate, appIdList, appType)
	gin_util.Response(ctx, resp, err)
}

// GetAppStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		获取应用统计列表
//	@Description	获取应用统计列表（分页）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			startDate	query		string	true	"开始时间（格式yyyy-mm-dd）"
//	@Param			endDate		query		string	true	"结束时间（格式yyyy-mm-dd）"
//	@Param			apps		query		string	false	"应用ID列表"
//	@Param			appType		query		string	false	"应用类型（默认agent）"
//	@Param			pageNo		query		int		true	"页面编号，从1开始"
//	@Param			pageSize	query		int		true	"单页数量"
//	@Success		200			{object}	response.Response{data=response.PageResult{list=[]response.AppStatisticItem}}
//	@Router			/statistic/app/list [get]
func GetAppStatisticList(ctx *gin.Context) {
	startDate := ctx.Query("startDate")
	endDate := ctx.Query("endDate")
	appIds := ctx.Query("apps")
	appType := ctx.Query("appType")
	var appIdList []string
	if appIds != "" {
		appIdList = strings.Split(appIds, ",")
	}
	resp, err := service.GetAppStatisticList(ctx, getUserID(ctx), getOrgID(ctx), startDate, endDate, appIdList, appType, getPageNo(ctx), getPageSize(ctx))
	gin_util.Response(ctx, resp, err)
}

// ExportAppStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		导出应用统计列表
//	@Description	导出应用统计列表数据
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			startDate	query		string	true	"开始时间（格式yyyy-mm-dd）"
//	@Param			endDate		query		string	true	"结束时间（格式yyyy-mm-dd）"
//	@Param			apps		query		string	false	"应用ID列表"
//	@Param			appType		query		string	false	"应用类型（默认agent）"
//	@Success		200			{object}	response.Response
//	@Router			/statistic/app/export [get]
func ExportAppStatisticList(ctx *gin.Context) {
	startDate := ctx.Query("startDate")
	endDate := ctx.Query("endDate")
	appIds := ctx.Query("apps")
	appType := ctx.Query("appType")
	var appIdList []string
	if appIds != "" {
		appIdList = strings.Split(appIds, ",")
	}
	file, err := service.ExportAppStatisticList(ctx, getUserID(ctx), getOrgID(ctx), startDate, endDate, appIdList, appType)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	fileName := fmt.Sprintf("应用统计列表_%v-%v.xlsx", startDate, endDate)
	ctx.Writer.WriteHeader(http.StatusOK)
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	if _, err := file.WriteTo(ctx.Writer); err != nil {
		gin_util.Response(ctx, nil, err)
	}
}

// GetAppListSelect
//
//	@Tags			app_observability.statistic
//	@Summary		获取当前用户在当前组织下发布的应用列表
//	@Description	获取当前用户在当前组织下发布的应用列表（包括私有发布、组织内发布、公开发布）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			appType	query		string	true	"应用类型"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.MyAppItem}}
//	@Router			/statistic/app/select [get]
func GetAppListSelect(ctx *gin.Context) {
	resp, err := service.GetAppListSelect(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("appType"))
	gin_util.Response(ctx, resp, err)
}
