package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"
)

var locationMu sync.Mutex

type Location struct {
	ID           uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt    time.Time      `json:"-"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedAtStr string         `gorm:"-" json:"createdAt"`
	UpdatedAtStr string         `gorm:"-" json:"updatedAt"`
	DeletedAtStr string         `gorm:"-" json:"deletedAt,omitempty"`
	IP           string         `json:"ip" gorm:"not null"`
	ISP          string         `json:"isp"`
	Country      string         `json:"country"`
	Province     string         `json:"province"`
	City         string         `json:"city"`
	Area         string         `json:"area"`
	Latitude     string         `json:"latitude"`
	Longitude    string         `json:"longitude"`
	Remark       string         `json:"remark"`
	LogCount     int64          `json:"logCount" gorm:"->;column:log_count"`
	Logs         []Log          `json:"logs,omitempty"`
}

func (l *Location) AfterFind(tx *gorm.DB) error {
	cst, _ := time.LoadLocation("Asia/Shanghai")
	if !l.CreatedAt.IsZero() {
		l.CreatedAtStr = l.CreatedAt.In(cst).Format("2006-01-02 15:04:05")
	}
	if !l.UpdatedAt.IsZero() {
		l.UpdatedAtStr = l.UpdatedAt.In(cst).Format("2006-01-02 15:04:05")
	}
	if l.DeletedAt.Valid {
		l.DeletedAtStr = l.DeletedAt.Time.In(cst).Format("2006-01-02 15:04:05")
	}
	return nil
}

func CreateLocation(l *Location) error {
	return DB.Create(l).Error
}

func GetLocationByID(id uint) (*Location, error) {
	var l Location
	err := DB.Preload("Logs").First(&l, id).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func GetLocationByIP(ip string) (*Location, error) {
	var l Location
	err := DB.Preload("Logs").Where("ip = ?", ip).First(&l).Error
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func fetchGeo(ip string) (country, city, isp, lat, lng, province, area string) {
	resp, err := http.Get(fmt.Sprintf("https://ip9.com.cn/get?ip=%s", ip))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var geo struct {
		Ret  int `json:"ret"`
		Data struct {
			Country string `json:"country"`
			City    string `json:"city"`
			Lat     string `json:"lat"`
			Lng     string `json:"lng"`
			ISP     string `json:"isp"`
			Prov    string `json:"prov"`
			Area    string `json:"area"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &geo) != nil || geo.Ret != 200 {
		fmt.Printf("解析出错 %s", string(body))
		return
	}
	return geo.Data.Country, geo.Data.City, geo.Data.ISP, geo.Data.Lat, geo.Data.Lng, geo.Data.Prov, geo.Data.Area
}

func GetOrCreateLocationByIP(ip string) (uint, error) {
	locationMu.Lock()
	var l Location
	err := DB.Where("ip = ?", ip).First(&l).Error
	if err == gorm.ErrRecordNotFound {
		l = Location{IP: ip}
		if err := DB.Create(&l).Error; err != nil {
			locationMu.Unlock()
			return 0, err
		}
	} else if err != nil {
		locationMu.Unlock()
		return 0, err
	}
	locationMu.Unlock()

	if l.ISP != "" || l.Country != "" {
		DB.Model(&l).Update("updated_at", time.Now())
		return l.ID, nil
	}

	country, city, isp, lat, lng, province, area := fetchGeo(ip)
	if isp == "" && country == "" {
		return l.ID, nil
	}

	l.Country = country
	l.City = city
	l.ISP = isp
	l.Latitude = lat
	l.Longitude = lng
	l.Province = province
	l.Area = area
	DB.Model(&l).Updates(map[string]any{
		"country":  country,
		"city":     city,
		"isp":      isp,
		"latitude": lat,
		"longitude": lng,
		"province": province,
		"area":     area,
	})
	return l.ID, nil
}

func UpdateLocation(l *Location) error {
	return DB.Save(l).Error
}

func UpdateLocationRemark(id uint, remark string) error {
	return DB.Model(&Location{}).Where("id = ?", id).Update("remark", remark).Error
}

func DeleteLocation(id uint) error {
	return DB.Delete(&Location{}, id).Error
}

func ListLocationsOrderByUpdatedAt(offset, limit int) ([]Location, int64, error) {
	var locations []Location
	var total int64
	if err := DB.Model(&Location{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Table("locations").
		Select(`locations.id, locations.created_at, locations.updated_at, locations.deleted_at,
			locations.ip, locations.isp, locations.country, locations.province, locations.city, locations.area,
			locations.latitude, locations.longitude, locations.remark,
			(SELECT COUNT(*) FROM logs WHERE logs.location_id = locations.id AND logs.deleted_at IS NULL AND DATE(logs.created_at) = CURRENT_DATE) AS log_count`).
		Offset(offset).Limit(limit).
		Order("updated_at DESC").
		Find(&locations).Error
	return locations, total, err
}

func ListLocations(offset, limit int) ([]Location, int64, error) {
	var locations []Location
	var total int64
	if err := DB.Model(&Location{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Table("locations").
		Select(`locations.id, locations.created_at, locations.updated_at, locations.deleted_at,
			locations.ip, locations.isp, locations.country, locations.province, locations.city, locations.area,
			locations.latitude, locations.longitude, locations.remark,
			(SELECT COUNT(*) FROM logs WHERE logs.location_id = locations.id AND logs.deleted_at IS NULL AND DATE(logs.created_at) = CURRENT_DATE) AS log_count`).
		Offset(offset).Limit(limit).
		Order("id DESC").
		Find(&locations).Error
	return locations, total, err
}
