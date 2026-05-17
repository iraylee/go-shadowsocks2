package model

import (
	"time"

	"gorm.io/gorm"
)

type Log struct {
	ID         uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"deletedAt" gorm:"index"`
	IP         string         `json:"ip" gorm:"not null"`
	URL        string         `json:"url" gorm:"not null"`
	LocationID uint           `json:"locationId"`
}

func CreateLog(l *Log) error {
	return DB.Create(l).Error
}

func GetLogByID(id uint) (*Log, error) {
	var l Log
	err := DB.First(&l, id).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func UpdateLog(l *Log) error {
	return DB.Save(l).Error
}

func DeleteLog(id uint) error {
	return DB.Delete(&Log{}, id).Error
}

func ListLogs(page int) ([]Log, int64, error) {
	cst, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(cst)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cst)
	end := day.AddDate(0, 0, -7*(page-1)+1)
	start := day.AddDate(0, 0, -7*page+1)

	var logs []Log
	var total int64
	if err := DB.Model(&Log{}).Where("created_at >= ? AND created_at < ?", start, end).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Where("created_at >= ? AND created_at < ?", start, end).Order("created_at DESC").Find(&logs).Error
	return logs, total, err
}

func ListLogsByIP(ip string, offset, limit int) ([]Log, int64, error) {
	var logs []Log
	var total int64
	if err := DB.Model(&Log{}).Where("ip = ?", ip).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Where("ip = ?", ip).Offset(offset).Limit(limit).Order("id DESC").Find(&logs).Error
	return logs, total, err
}

func ListLogsByLocationID(locationID uint, offset, limit int) ([]Log, int64, error) {
	var logs []Log
	var total int64
	if err := DB.Model(&Log{}).Where("location_id = ?", locationID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Where("location_id = ?", locationID).Offset(offset).Limit(limit).Order("id DESC").Find(&logs).Error
	return logs, total, err
}

func CountLogsByDateByHour(ip string, date time.Time) ([]int64, error) {
	startOfDay := date
	endOfDay := startOfDay.AddDate(0, 0, 1)

	var results []struct {
		Hour  int
		Count int64
	}

	tz := date.Location().String()
	query := DB.Raw(
		"SELECT EXTRACT(HOUR FROM created_at AT TIME ZONE ?)::int AS hour, COUNT(*) AS count FROM logs WHERE created_at >= ? AND created_at < ? GROUP BY hour ORDER BY hour",
		tz, startOfDay, endOfDay,
	)
	if ip != "" {
		query = DB.Raw(
			"SELECT EXTRACT(HOUR FROM created_at AT TIME ZONE ?)::int AS hour, COUNT(*) AS count FROM logs WHERE ip = ? AND created_at >= ? AND created_at < ? GROUP BY hour ORDER BY hour",
			tz, ip, startOfDay, endOfDay,
		)
	}
	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	counts := make([]int64, 24)
	for _, r := range results {
		if r.Hour >= 0 && r.Hour < 24 {
			counts[r.Hour] = r.Count
		}
	}
	return counts, nil
}
