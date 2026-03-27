package orm

import (
	"context"
	"fmt"
	"time"

	"github.com/UnicomAI/wanwu/internal/app-service/client/model"
	"github.com/UnicomAI/wanwu/internal/app-service/client/orm/sqlopt"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const (
	cronTaskStatisticSync = "CronTaskStatisticSync"
)

var statisticCronManager *StatisticCronManager

type StatisticCronManager struct {
	ctx  context.Context
	cron *cron.Cron
	db   *gorm.DB
}

func CronInit(ctx context.Context, db *gorm.DB) error {
	statisticCronManager = &StatisticCronManager{
		ctx:  ctx,
		cron: cron.New(),
		db:   db,
	}

	if err := syncAllStatistics(); err != nil {
		return fmt.Errorf("sync statistics err: %v", err)
	}

	entryID, err := statisticCronManager.cron.AddFunc("0 * * * *", cronSyncAllStatistics) // 每小时整点执行
	if err != nil {
		log.Errorf("register cron task (%v) error: %v", cronTaskStatisticSync, err)
		return err
	}
	log.Infof("cron task (%v) registered with entry ID: %d", cronTaskStatisticSync, entryID)

	statisticCronManager.cron.Start()

	return nil
}

func CronStop() {
	if statisticCronManager != nil {
		statisticCronManager.cron.Stop()
		log.Infof("cron tasks stopped")
	}
}

func cronSyncAllStatistics() {
	defer util.PrintPanicStack()
	if err := syncAllStatistics(); err != nil {
		log.Errorf("execute statistics sync err: %v", err)
	}
}

func syncAllStatistics() error {
	ctx := statisticCronManager.ctx
	db := statisticCronManager.db

	now := time.Now().UnixMilli()
	startTs := now - 30*24*time.Hour.Milliseconds()
	dates := util.DateRange(startTs, now)

	for i := len(dates) - 1; i >= 0; i-- {
		date := dates[i]
		// 检查MySQL中是否已有该日期的统计数据
		// 如果存在，说明历史数据已同步，无需继续向前回填
		hasModel, _ := checkModelStatsRecordExists(ctx, db, date)
		hasApp, _ := checkAppStatsRecordExists(ctx, db, date)
		hasApi, _ := checkAPIKeyStatsRecordExists(ctx, db, date)
		if err := updateModelStats(ctx, date, db); err != nil {
			log.Errorf("update model stats date %v err: %v", date, err)
		}
		if err := updateAppStats(ctx, date, db); err != nil {
			log.Errorf("update app stats date %v err: %v", date, err)
		}
		if err := updateAPIKeyStats(ctx, date, db); err != nil {
			log.Errorf("update api key stats date %v err: %v", date, err)
		}
		if hasModel && hasApp && hasApi {
			log.Infof("found existing record for date %v, stop backward sync", date)
			break
		}
	}
	return nil
}

func checkModelStatsRecordExists(ctx context.Context, db *gorm.DB, date string) (bool, error) {
	var count int64
	if err := sqlopt.SQLOptions(
		sqlopt.WithDate(date),
	).Apply(db.WithContext(ctx)).Model(&model.ModelRecord{}).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check model stats record exists for date %v err: %v", date, err)
	}
	return count > 0, nil
}

func checkAppStatsRecordExists(ctx context.Context, db *gorm.DB, date string) (bool, error) {
	var count int64
	if err := sqlopt.SQLOptions(
		sqlopt.WithDate(date),
	).Apply(db.WithContext(ctx)).Model(&model.AppRecord{}).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check app stats record exists for date %v err: %v", date, err)
	}
	return count > 0, nil
}

// checkAPIKeyStatsRecordExists 检查指定日期的 API Key 统计记录是否已存在
func checkAPIKeyStatsRecordExists(ctx context.Context, db *gorm.DB, date string) (bool, error) {
	var count int64
	if err := sqlopt.SQLOptions(
		sqlopt.WithDate(date),
	).Apply(db.WithContext(ctx)).Model(&model.APIKeyStatistic{}).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check api key stats record exists for date %v err: %v", date, err)
	}
	return count > 0, nil
}
