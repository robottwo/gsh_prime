package analytics

import (
	"fmt"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"mvdan.cc/sh/v3/interp"
)

type AnalyticsManager struct {
	db     *gorm.DB
	Runner *interp.Runner
	Logger *zap.Logger
}

type AnalyticsEntry struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`

	Input      string
	Prediction string
	Actual     string
}

func NewAnalyticsManager(dbFilePath string) (*AnalyticsManager, error) {
	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database")
		return nil, err
	}

	if err := db.AutoMigrate(&AnalyticsEntry{}); err != nil { return nil, err }

	return &AnalyticsManager{
		db: db,
	}, nil
}

func (analyticsManager *AnalyticsManager) NewEntry(input string, prediction string, actual string) error {
	entry := AnalyticsEntry{
		Input:      input,
		Prediction: prediction,
		Actual:     actual,
	}

	result := analyticsManager.db.Create(&entry)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (analyticsManager *AnalyticsManager) GetRecentEntries(limit int) ([]AnalyticsEntry, error) {
	var entries []AnalyticsEntry
	result := analyticsManager.db.Where("input <> '' AND actual NOT LIKE '#%'").Order("created_at desc").Limit(limit).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}

func (analyticsManager *AnalyticsManager) ResetAnalytics() error {
	result := analyticsManager.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&AnalyticsEntry{})
	return result.Error
}

func (analyticsManager *AnalyticsManager) DeleteEntry(id uint) error {
	result := analyticsManager.db.Delete(&AnalyticsEntry{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("entry not found")
	}
	return nil
}

func (analyticsManager *AnalyticsManager) GetTotalCount() (int64, error) {
	var count int64
	result := analyticsManager.db.Model(&AnalyticsEntry{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (analyticsManager *AnalyticsManager) GetAllEntries() ([]AnalyticsEntry, error) {
	var entries []AnalyticsEntry
	result := analyticsManager.db.Where("input <> '' AND actual NOT LIKE '#%'").Order("created_at asc").Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}

func (analyticsManager *AnalyticsManager) GetCommandFrequencies() (map[string]int64, error) {
	var results []struct {
		Actual string
		Count  int64
	}
	if err := analyticsManager.db.Model(&AnalyticsEntry{}).Select("actual, count(*) as count").Where("actual NOT LIKE '#%'").Group("actual").Scan(&results).Error; err != nil {
		return nil, err
	}

	freq := make(map[string]int64)
	for _, r := range results {
		freq[r.Actual] = r.Count
	}
	return freq, nil
}

// GetDailyActivity returns a map of date string (YYYY-MM-DD) to command count
func (analyticsManager *AnalyticsManager) GetDailyActivity() (map[string]int64, error) {
	var results []struct {
		Date  string
		Count int64
	}
	// SQLite specific date function
	if err := analyticsManager.db.Model(&AnalyticsEntry{}).Select("date(created_at) as date, count(*) as count").Group("date(created_at)").Scan(&results).Error; err != nil {
		return nil, err
	}

	activity := make(map[string]int64)
	for _, r := range results {
		activity[r.Date] = r.Count
	}
	return activity, nil
}
