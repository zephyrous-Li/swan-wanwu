package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/app-service/client/model"
	"github.com/UnicomAI/wanwu/internal/app-service/client/orm/sqlopt"
	"github.com/UnicomAI/wanwu/pkg/constant"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/redis"
	"github.com/UnicomAI/wanwu/pkg/util"
	"gorm.io/gorm"
)

const luaUpdateAppStats = `
local key = KEYS[1]
local field = KEYS[2]
local delta = cjson.decode(ARGV[1])
local expire = tonumber(ARGV[2])

local current = redis.call('HGET', key, field)

if current then
	local record = cjson.decode(current)
	record.callCount = record.callCount + delta.callCount
	record.callFailure = record.callFailure + delta.callFailure
	record.streamCount = record.streamCount + delta.streamCount
	record.streamFailure = record.streamFailure + delta.streamFailure
	record.streamCosts = record.streamCosts + delta.streamCosts
	record.nonStreamCount = record.nonStreamCount + delta.nonStreamCount
	record.nonStreamFailure = record.nonStreamFailure + delta.nonStreamFailure
	record.nonStreamCosts = record.nonStreamCosts + delta.nonStreamCosts
	record.webCallCount = record.webCallCount + delta.webCallCount
	record.webCallFailure = record.webCallFailure + delta.webCallFailure
	record.openapiCallCount = record.openapiCallCount + delta.openapiCallCount
	record.openapiCallFailure = record.openapiCallFailure + delta.openapiCallFailure
	record.webUrlCallCount = record.webUrlCallCount + delta.webUrlCallCount
	record.webUrlCallFailure = record.webUrlCallFailure + delta.webUrlCallFailure
	redis.call('HSET', key, field, cjson.encode(record))
else
	redis.call('HSET', key, field, ARGV[1])
end

redis.call('EXPIRE', key, expire)
return 1
`

type AppRecordStats struct {
	AppType            string `json:"appType"`
	CallCount          int32  `json:"callCount"`
	CallFailure        int32  `json:"callFailure"`
	StreamCount        int32  `json:"streamCount"`
	StreamFailure      int32  `json:"streamFailure"`
	StreamCosts        int64  `json:"streamCosts"`
	NonStreamCount     int32  `json:"nonStreamCount"`
	NonStreamFailure   int32  `json:"nonStreamFailure"`
	NonStreamCosts     int64  `json:"nonStreamCosts"`
	WebCallCount       int32  `json:"webCallCount"` //网页调用次数
	WebCallFailure     int32  `json:"webCallFailure"`
	OpenapiCallCount   int32  `json:"openapiCallCount"` // openapi调用次数
	OpenapiCallFailure int32  `json:"openapiCallFailure"`
	WebUrlCallCount    int32  `json:"webUrlCallCount"` // 仅对于智能体应用统计
	WebUrlCallFailure  int32  `json:"webUrlCallFailure"`
}

// GetAppStatistic 获取应用统计（概览+趋势）
func (c *Client) GetAppStatistic(ctx context.Context, userId, orgId, startDate, endDate string, appIds []string, appType string) (*AppStatistic, *errs.Status) {
	if startDate > endDate {
		return nil, toErrStatus("app_statistic_get", fmt.Errorf("startDate %v greater than endDate %v", startDate, endDate).Error())
	}

	today := util.Time2Date(time.Now().UnixMilli())
	if err := updateAppStats(ctx, today, c.db); err != nil {
		log.Errorf("sync app stats for today %v err: %v", today, err)
	}

	overview, err := statisticAppStatsOverview(ctx, c.db, userId, orgId, startDate, endDate, appIds, appType)
	if err != nil {
		return nil, toErrStatus("app_statistic_get", err.Error())
	}

	trend, err := statisticAppStatsTrend(ctx, c.db, userId, orgId, startDate, endDate, appIds, appType)
	if err != nil {
		return nil, toErrStatus("app_statistic_get", err.Error())
	}

	return &AppStatistic{
		Overview: *overview,
		Trend:    *trend,
	}, nil
}

// GetAppStatisticList 获取应用统计列表（分页）
func (c *Client) GetAppStatisticList(ctx context.Context, userId, orgId, startDate, endDate string, appIds []string, appType string, offset, limit int32) (*AppStatisticList, *errs.Status) {
	if startDate > endDate {
		return nil, toErrStatus("app_statistic_list_get", fmt.Errorf("startDate %v greater than endDate %v", startDate, endDate).Error())
	}

	today := util.Time2Date(time.Now().UnixMilli())
	if err := updateAppStats(ctx, today, c.db); err != nil {
		log.Errorf("sync app stats for today %v err: %v", today, err)
	}

	items, total, err := getAppStatisticList(ctx, c.db, userId, orgId, startDate, endDate, appIds, appType, offset, limit)
	if err != nil {
		return nil, toErrStatus("app_statistic_list_get", err.Error())
	}

	return &AppStatisticList{
		Items: items,
		Total: total,
	}, nil
}

// RecordAppStatistic 记录应用统计数据
func (c *Client) RecordAppStatistic(ctx context.Context, userId, orgId, appId, appType string, isSuccess, isStream bool, streamCosts, nonStreamCosts int64, source string) *errs.Status {
	err := recordAppStatistic(ctx, userId, orgId, appId, appType, isSuccess, isStream, streamCosts, nonStreamCosts, source)
	if err != nil {
		return toErrStatus("app_record_statistic", err.Error())
	}
	return nil
}

func getRedisAppStatsKey(date string) string {
	return fmt.Sprintf("apprecord|%s", date)
}

func getRedisAppStatsItemField(appId, appType, userId, orgId string) string {
	return fmt.Sprintf("%s|%s|%s|%s", appId, appType, userId, orgId)
}

func parseRedisAppStatsItem(field string) (appId, appType, userId, orgId string, ok bool) {
	parts := strings.Split(field, "|")
	if len(parts) != 4 {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}

func getRedisAppStatsItemValue(value string) (*AppRecordStats, error) {
	record := &AppRecordStats{}
	if err := json.Unmarshal([]byte(value), record); err != nil {
		return nil, fmt.Errorf("unmarshal value %v err: %v", value, err)
	}
	return record, nil
}

func recordAppStatistic(ctx context.Context, userId, orgId, appId, appType string, isSuccess, isStream bool, streamCosts, nonStreamCosts int64, source string) error {
	today := util.Time2Date(time.Now().UnixMilli())
	key := getRedisAppStatsKey(today)
	field := getRedisAppStatsItemField(appId, appType, userId, orgId)

	callFailure := 0
	streamCount := 0
	streamFailure := 0
	nonStreamCount := 0
	nonStreamFailure := 0
	webCallCount := 0
	webCallFailure := 0
	openapiCallCount := 0
	openapiCallFailure := 0
	webUrlCallCount := 0
	webUrlCallFailure := 0

	if isStream {
		streamCount = 1
		if !isSuccess {
			streamFailure = 1
		}
	} else {
		nonStreamCount = 1
		if !isSuccess {
			nonStreamFailure = 1
		}
	}
	if !isSuccess {
		callFailure = 1
	}

	switch source {
	case constant.AppStatisticSourceWeb:
		webCallCount = 1
		if !isSuccess {
			webCallFailure = 1
		}
	case constant.AppStatisticSourceOpenAPI:
		openapiCallCount = 1
		if !isSuccess {
			openapiCallFailure = 1
		}
	case constant.AppStatisticSourceWebUrl:
		webUrlCallCount = 1
		if !isSuccess {
			webUrlCallFailure = 1
		}
	}

	delta := map[string]any{
		"appType":            appType,
		"callCount":          1,
		"callFailure":        callFailure,
		"streamCount":        streamCount,
		"streamFailure":      streamFailure,
		"streamCosts":        streamCosts,
		"nonStreamCount":     nonStreamCount,
		"nonStreamFailure":   nonStreamFailure,
		"nonStreamCosts":     nonStreamCosts,
		"webCallCount":       webCallCount,
		"webCallFailure":     webCallFailure,
		"openapiCallCount":   openapiCallCount,
		"openapiCallFailure": openapiCallFailure,
		"webUrlCallCount":    webUrlCallCount,
		"webUrlCallFailure":  webUrlCallFailure,
	}

	deltaJSON, _ := json.Marshal(delta)

	_, err := redis.App().Eval(ctx, luaUpdateAppStats, []string{key, field}, string(deltaJSON), redisStatsExpireSeconds)
	if err != nil {
		return fmt.Errorf("redis eval err: %v", err)
	}
	return nil
}

func updateAppStats(ctx context.Context, date string, db *gorm.DB) error {
	key := getRedisAppStatsKey(date)
	resultMap, err := redis.App().HGetAll(ctx, key)
	if err != nil {
		return fmt.Errorf("redis HGetAll key %v failed: %v", key, err)
	}
	if len(resultMap) == 0 {
		return nil
	}

	for _, item := range resultMap {
		if appId, appType, userId, orgId, ok := parseRedisAppStatsItem(item.K); ok {
			record, err := getRedisAppStatsItemValue(item.V)
			if err != nil {
				log.Errorf("get app stat item %v err: %v", item.K, err)
				continue
			}
			if err := updateAppStatsByRecord(ctx, db, appId, appType, userId, orgId, date, record); err != nil {
				log.Errorf("update date %v app failed for appId %v userId %v orgId %v err: %v",
					date, appId, userId, orgId, err)
			}
		}
	}
	return nil
}

func updateAppStatsByRecord(ctx context.Context, db *gorm.DB, appId, appType, userId, orgId, date string, record *AppRecordStats) error {
	appStat := &model.AppRecord{
		OrgID:   orgId,
		UserID:  userId,
		AppID:   appId,
		AppType: appType,
		Date:    date,
	}

	if err := db.WithContext(ctx).Where(&model.AppRecord{
		OrgID:   orgId,
		UserID:  userId,
		AppID:   appId,
		AppType: appType,
		Date:    date,
	}).FirstOrCreate(appStat).Error; err != nil {
		return err
	}

	if appStat.CallCount == record.CallCount &&
		appStat.CallFailure == record.CallFailure &&
		appStat.StreamCount == record.StreamCount &&
		appStat.StreamFailure == record.StreamFailure &&
		appStat.StreamCosts == record.StreamCosts &&
		appStat.NonStreamCount == record.NonStreamCount &&
		appStat.NonStreamFailure == record.NonStreamFailure &&
		appStat.NonStreamCosts == record.NonStreamCosts &&
		appStat.WebCallCount == record.WebCallCount &&
		appStat.WebCallFailure == record.WebCallFailure &&
		appStat.OpenapiCallCount == record.OpenapiCallCount &&
		appStat.OpenapiCallFailure == record.OpenapiCallFailure &&
		appStat.WebUrlCallCount == record.WebUrlCallCount &&
		appStat.WebUrlCallFailure == record.WebUrlCallFailure {
		return nil
	}

	return db.WithContext(ctx).Model(appStat).Updates(map[string]any{
		"call_count":           record.CallCount,
		"call_failure":         record.CallFailure,
		"stream_count":         record.StreamCount,
		"stream_failure":       record.StreamFailure,
		"stream_costs":         record.StreamCosts,
		"non_stream_count":     record.NonStreamCount,
		"non_stream_failure":   record.NonStreamFailure,
		"non_stream_costs":     record.NonStreamCosts,
		"web_call_count":       record.WebCallCount,
		"web_call_failure":     record.WebCallFailure,
		"openapi_call_count":   record.OpenapiCallCount,
		"openapi_call_failure": record.OpenapiCallFailure,
		"web_url_call_count":   record.WebUrlCallCount,
		"web_url_call_failure": record.WebUrlCallFailure,
	}).Error
}

func getAppStatisticList(ctx context.Context, db *gorm.DB, userId, orgId, startDate, endDate string, appIds []string, appType string, offset, limit int32) ([]AppStatisticItem, int32, error) {
	opts := []sqlopt.SQLOption{
		sqlopt.WithUserID(userId),
		sqlopt.WithOrgID(orgId),
		sqlopt.StartDate(startDate),
		sqlopt.EndDate(endDate),
		sqlopt.WithAppType(appType),
		sqlopt.WithAppIDsForStatistic(appIds),
	}
	var total int64
	countQuery := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).
		Model(&model.AppRecord{}).
		Select("COUNT(DISTINCT app_id)")
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count app stat list err: %v", err)
	}
	var stats []model.AppRecord
	query := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).
		Select("app_id, ANY_VALUE(app_type) as app_type, " +
			"ANY_VALUE(org_id) as org_id, " +
			"SUM(call_count) as call_count, SUM(call_failure) as call_failure, " +
			"SUM(stream_count) as stream_count, SUM(stream_failure) as stream_failure, " +
			"SUM(stream_costs) as stream_costs, " +
			"SUM(non_stream_count) as non_stream_count, SUM(non_stream_failure) as non_stream_failure, " +
			"SUM(non_stream_costs) as non_stream_costs").
		Group("app_id").Order("call_count DESC").Offset(int(offset)).Limit(int(limit))

	if err := query.Find(&stats).Error; err != nil {
		return nil, 0, fmt.Errorf("get app stat list err: %v", err)
	}

	items := make([]AppStatisticItem, 0, len(stats))
	for _, stat := range stats {
		failureRate := calculateFailureRate(stat.CallFailure, stat.CallCount)
		avgStreamCosts := calculateAvg(stat.StreamCosts, calculateSuccessCount(stat.StreamCount, stat.StreamFailure))
		avgNonStreamCosts := calculateAvg(stat.NonStreamCosts, calculateSuccessCount(stat.NonStreamCount, stat.NonStreamFailure))
		items = append(items, AppStatisticItem{
			AppId:             stat.AppID,
			AppType:           stat.AppType,
			OrgId:             stat.OrgID,
			CallCount:         stat.CallCount,
			CallFailure:       stat.CallFailure,
			FailureRate:       failureRate,
			StreamCount:       stat.StreamCount,
			NonStreamCount:    stat.NonStreamCount,
			AvgStreamCosts:    avgStreamCosts,
			AvgNonStreamCosts: avgNonStreamCosts,
		})
	}

	return items, int32(total), nil
}

func statisticAppStatsOverview(ctx context.Context, db *gorm.DB, userID, orgID, startDate, endDate string, appIds []string, appType string) (*AppStatisticOverview, error) {
	prevPeriod, currPeriod, err := util.PreviousDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	current, err := appStatsByDateRange(ctx, db, userID, orgID, currPeriod, appIds, appType)
	if err != nil {
		return nil, err
	}

	previous, err := appStatsByDateRange(ctx, db, userID, orgID, prevPeriod, appIds, appType)
	if err != nil {
		return nil, err
	}

	current.CallCount.PeriodOverPeriod = calculatePoP(current.CallCount.Value, previous.CallCount.Value)
	current.CallFailure.PeriodOverPeriod = calculatePoP(current.CallFailure.Value, previous.CallFailure.Value)
	current.StreamCount.PeriodOverPeriod = calculatePoP(current.StreamCount.Value, previous.StreamCount.Value)
	current.NonStreamCount.PeriodOverPeriod = calculatePoP(current.NonStreamCount.Value, previous.NonStreamCount.Value)
	current.AvgStreamCosts.PeriodOverPeriod = calculatePoP(current.AvgStreamCosts.Value, previous.AvgStreamCosts.Value)
	current.AvgNonStreamCosts.PeriodOverPeriod = calculatePoP(current.AvgNonStreamCosts.Value, previous.AvgNonStreamCosts.Value)

	return current, nil
}

func appStatsByDateRange(ctx context.Context, db *gorm.DB, userID, orgID string, dates []string, appIds []string, appType string) (*AppStatisticOverview, error) {
	startDate, endDate := dates[0], dates[len(dates)-1]
	var stat model.AppRecord
	opts := []sqlopt.SQLOption{
		sqlopt.WithOrgID(orgID),
		sqlopt.WithUserID(userID),
		sqlopt.StartDate(startDate),
		sqlopt.EndDate(endDate),
		sqlopt.WithAppIDsForStatistic(appIds),
		sqlopt.WithAppType(appType),
	}
	if err := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).
		Select("SUM(call_count) as call_count, " +
			"SUM(call_failure) as call_failure, " +
			"SUM(stream_count) as stream_count, " +
			"SUM(stream_failure) as stream_failure, " +
			"SUM(stream_costs) as stream_costs, " +
			"SUM(non_stream_count) as non_stream_count, " +
			"SUM(non_stream_failure) as non_stream_failure, " +
			"SUM(non_stream_costs) as non_stream_costs").
		First(&stat).Error; err != nil {
		return nil, fmt.Errorf("app stat [%v, %v] err: %v", startDate, endDate, err)
	}

	avgStreamCosts := calculateAvg(stat.StreamCosts, calculateSuccessCount(stat.StreamCount, stat.StreamFailure))
	avgNonStreamCosts := calculateAvg(stat.NonStreamCosts, calculateSuccessCount(stat.NonStreamCount, stat.NonStreamFailure))

	return &AppStatisticOverview{
		CallCount:         StatisticOverviewItem{Value: float32(stat.CallCount)},
		CallFailure:       StatisticOverviewItem{Value: float32(stat.CallFailure)},
		StreamCount:       StatisticOverviewItem{Value: float32(stat.StreamCount)},
		NonStreamCount:    StatisticOverviewItem{Value: float32(stat.NonStreamCount)},
		AvgStreamCosts:    StatisticOverviewItem{Value: avgStreamCosts},
		AvgNonStreamCosts: StatisticOverviewItem{Value: avgNonStreamCosts},
	}, nil
}

func statisticAppStatsTrend(ctx context.Context, db *gorm.DB, userID, orgID, startDate, endDate string, appIds []string, appType string) (*AppStatisticTrend, error) {
	dates, err := BuildDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	var stats []model.AppRecord
	opts := []sqlopt.SQLOption{
		sqlopt.WithUserID(userID),
		sqlopt.WithOrgID(orgID),
		sqlopt.StartDate(startDate),
		sqlopt.EndDate(endDate),
		sqlopt.WithAppIDsForStatistic(appIds),
		sqlopt.WithAppType(appType),
	}
	if err := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).
		Select("date, SUM(call_count) as call_count, SUM(call_failure) as call_failure, " +
			"SUM(stream_count) as stream_count, SUM(stream_failure) as stream_failure, " +
			"SUM(stream_costs) as stream_costs, " +
			"SUM(non_stream_count) as non_stream_count, SUM(non_stream_failure) as non_stream_failure, " +
			"SUM(non_stream_costs) as non_stream_costs, " +
			"SUM(web_call_count) as web_call_count, " +
			"SUM(openapi_call_count) as openapi_call_count, " +
			"SUM(web_url_call_count) as web_url_call_count").
		Group("date").
		Order("date").
		Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("app stat trend err: %v", err)
	}

	callTrendLineNames := []string{
		"app_statistic_call_count_total",
		"app_statistic_web_call_count",
		"app_statistic_openapi_call_count",
		"app_statistic_web_url_call_count",
	}
	callTrendLines := BuildChartLines(stats, dates,
		func(r model.AppRecord) string { return r.Date },
		func(r model.AppRecord) map[string]float32 {
			return map[string]float32{
				"app_statistic_call_count_total":   float32(r.CallCount),
				"app_statistic_web_call_count":     float32(r.WebCallCount),
				"app_statistic_openapi_call_count": float32(r.OpenapiCallCount),
				"app_statistic_web_url_call_count": float32(r.WebUrlCallCount),
			}
		},
		callTrendLineNames,
	)

	return &AppStatisticTrend{
		CallTrend: StatisticChart{
			Name:  "app_statistic_call_trend",
			Lines: callTrendLines,
		},
	}, nil
}
