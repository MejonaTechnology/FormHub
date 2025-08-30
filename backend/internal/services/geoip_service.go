package services

import (
	"context"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/database"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
)

// GeoIPService handles geographic information lookup and caching
type GeoIPService struct {
	db    *sqlx.DB
	redis *database.RedisClient
	apiKey string // For external GeoIP services
}

// GeoIPResult represents geographic information for an IP address
type GeoIPResult struct {
	IP          string   `json:"ip"`
	CountryCode string   `json:"country_code"`
	CountryName string   `json:"country_name"`
	Region      string   `json:"region"`
	City        string   `json:"city"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	Timezone    string   `json:"timezone"`
	ISP         string   `json:"isp,omitempty"`
	ASN         string   `json:"asn,omitempty"`
}

// UserAgentInfo represents parsed user agent information
type UserAgentInfo struct {
	DeviceType     models.DeviceType `json:"device_type"`
	BrowserName    string            `json:"browser_name"`
	BrowserVersion string            `json:"browser_version"`
	OSName         string            `json:"os_name"`
	OSVersion      string            `json:"os_version"`
	IsMobile       bool              `json:"is_mobile"`
	IsBot          bool              `json:"is_bot"`
	BotName        string            `json:"bot_name,omitempty"`
}

func NewGeoIPService(db *sqlx.DB, redis *database.RedisClient, apiKey string) *GeoIPService {
	return &GeoIPService{
		db:     db,
		redis:  redis,
		apiKey: apiKey,
	}
}

// GetGeoInfo gets geographic information for an IP address
func (g *GeoIPService) GetGeoInfo(ctx context.Context, ipAddress string) (*GeoIPResult, error) {
	// Check if it's a local IP
	if g.isLocalIP(ipAddress) {
		return &GeoIPResult{
			IP:          ipAddress,
			CountryCode: "US",
			CountryName: "United States",
			Region:      "California",
			City:        "San Francisco",
			Timezone:    "America/Los_Angeles",
		}, nil
	}

	// Try cache first
	if cached := g.getFromCache(ctx, ipAddress); cached != nil {
		return cached, nil
	}

	// Get from external service
	result, err := g.lookupExternal(ctx, ipAddress)
	if err != nil {
		log.Printf("Failed to lookup IP %s: %v", ipAddress, err)
		// Return default/unknown values
		result = &GeoIPResult{
			IP:          ipAddress,
			CountryCode: "XX",
			CountryName: "Unknown",
			Timezone:    "UTC",
		}
	}

	// Cache the result
	g.cacheResult(ctx, ipAddress, result)

	return result, nil
}

// ParseUserAgent parses user agent string for device and browser information
func (g *GeoIPService) ParseUserAgent(userAgent string) *UserAgentInfo {
	info := &UserAgentInfo{
		DeviceType:     models.DeviceTypeUnknown,
		BrowserName:    "Unknown",
		BrowserVersion: "",
		OSName:         "Unknown",
		OSVersion:      "",
	}

	if userAgent == "" {
		return info
	}

	ua := strings.ToLower(userAgent)

	// Bot detection
	botPatterns := []string{
		"bot", "crawler", "spider", "scraper", "parser", "checker",
		"google", "bing", "yahoo", "duckduckbot", "facebook", "twitter",
		"linkedin", "pinterest", "instagram", "whatsapp", "telegram",
		"slackbot", "discordbot", "curl", "wget", "python-requests",
	}

	for _, pattern := range botPatterns {
		if strings.Contains(ua, pattern) {
			info.IsBot = true
			info.BotName = pattern
			break
		}
	}

	// Device type detection
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || 
	   strings.Contains(ua, "android") || strings.Contains(ua, "windows phone") {
		info.DeviceType = models.DeviceTypeMobile
		info.IsMobile = true
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		info.DeviceType = models.DeviceTypeTablet
	} else {
		info.DeviceType = models.DeviceTypeDesktop
	}

	// Browser detection
	info.BrowserName, info.BrowserVersion = g.detectBrowser(ua)

	// OS detection
	info.OSName, info.OSVersion = g.detectOS(ua)

	return info
}

// EnrichAnalyticsEvent enriches an analytics event with geographic and device info
func (g *GeoIPService) EnrichAnalyticsEvent(ctx context.Context, event *models.FormAnalyticsEvent) error {
	// Get geographic info
	if event.IPAddress != "" {
		geoInfo, err := g.GetGeoInfo(ctx, event.IPAddress)
		if err == nil {
			event.CountryCode = &geoInfo.CountryCode
			event.CountryName = &geoInfo.CountryName
			event.Region = &geoInfo.Region
			event.City = &geoInfo.City
			event.Latitude = geoInfo.Latitude
			event.Longitude = geoInfo.Longitude
			event.Timezone = &geoInfo.Timezone
		}
	}

	// Parse user agent
	if event.UserAgent != nil && *event.UserAgent != "" {
		uaInfo := g.ParseUserAgent(*event.UserAgent)
		
		event.DeviceType = uaInfo.DeviceType
		event.BrowserName = &uaInfo.BrowserName
		event.BrowserVersion = &uaInfo.BrowserVersion
		event.OSName = &uaInfo.OSName
		event.OSVersion = &uaInfo.OSVersion

		// Add device info to event data
		if event.EventData == nil {
			event.EventData = make(map[string]interface{})
		}
		event.EventData["is_mobile"] = uaInfo.IsMobile
		event.EventData["is_bot"] = uaInfo.IsBot
		if uaInfo.BotName != "" {
			event.EventData["bot_name"] = uaInfo.BotName
		}
	}

	return nil
}

// AggregateGeographicData aggregates geographic analytics for a form
func (g *GeoIPService) AggregateGeographicData(ctx context.Context, formID string, date time.Time) error {
	dateStr := date.Format("2006-01-02")

	query := `
		INSERT INTO form_geographic_analytics (
			id, form_id, user_id, date, country_code, country_name, region, city,
			total_views, total_submissions, conversion_rate, created_at, updated_at
		)
		SELECT 
			UUID() as id,
			form_id,
			user_id,
			DATE(created_at) as date,
			country_code,
			country_name,
			region,
			city,
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) as total_views,
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) as total_submissions,
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'form_view' THEN 1 END))
				ELSE 0 
			END as conversion_rate,
			NOW() as created_at,
			NOW() as updated_at
		FROM form_analytics_events 
		WHERE form_id = ? AND DATE(created_at) = ? 
		      AND country_code IS NOT NULL
		GROUP BY form_id, user_id, country_code, country_name, region, city
		ON DUPLICATE KEY UPDATE
			total_views = VALUES(total_views),
			total_submissions = VALUES(total_submissions),
			conversion_rate = VALUES(conversion_rate),
			updated_at = NOW()
	`

	_, err := g.db.ExecContext(ctx, query, formID, dateStr)
	return err
}

// AggregateDeviceData aggregates device analytics for a form
func (g *GeoIPService) AggregateDeviceData(ctx context.Context, formID string, date time.Time) error {
	dateStr := date.Format("2006-01-02")

	query := `
		INSERT INTO form_device_analytics (
			id, form_id, user_id, date, device_type, browser_name, browser_version, 
			os_name, os_version, total_views, total_submissions, conversion_rate, 
			created_at, updated_at
		)
		SELECT 
			UUID() as id,
			form_id,
			user_id,
			DATE(created_at) as date,
			device_type,
			browser_name,
			browser_version,
			os_name,
			os_version,
			COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) as total_views,
			COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) as total_submissions,
			CASE 
				WHEN COUNT(CASE WHEN event_type = 'form_view' THEN 1 END) > 0 
				THEN (COUNT(CASE WHEN event_type = 'form_submit' THEN 1 END) * 100.0 / COUNT(CASE WHEN event_type = 'form_view' THEN 1 END))
				ELSE 0 
			END as conversion_rate,
			NOW() as created_at,
			NOW() as updated_at
		FROM form_analytics_events 
		WHERE form_id = ? AND DATE(created_at) = ?
		      AND device_type IS NOT NULL AND browser_name IS NOT NULL
		GROUP BY form_id, user_id, device_type, browser_name, browser_version, os_name, os_version
		ON DUPLICATE KEY UPDATE
			total_views = VALUES(total_views),
			total_submissions = VALUES(total_submissions),
			conversion_rate = VALUES(conversion_rate),
			updated_at = NOW()
	`

	_, err := g.db.ExecContext(ctx, query, formID, dateStr)
	return err
}

// Private methods

func (g *GeoIPService) isLocalIP(ip string) bool {
	localIPs := []string{"127.0.0.1", "::1", "localhost"}
	for _, localIP := range localIPs {
		if ip == localIP {
			return true
		}
	}

	// Check for private IP ranges
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}

	// IPv4 private ranges
	private := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, privateRange := range private {
		_, cidr, _ := net.ParseCIDR(privateRange)
		if cidr != nil && cidr.Contains(parsed) {
			return true
		}
	}

	return false
}

func (g *GeoIPService) getFromCache(ctx context.Context, ip string) *GeoIPResult {
	key := fmt.Sprintf("geoip:%s", ip)
	data, err := g.redis.Client.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return nil
	}

	var result GeoIPResult
	if json.Unmarshal([]byte(data), &result) != nil {
		return nil
	}

	return &result
}

func (g *GeoIPService) cacheResult(ctx context.Context, ip string, result *GeoIPResult) {
	key := fmt.Sprintf("geoip:%s", ip)
	data, _ := json.Marshal(result)
	g.redis.Client.Set(ctx, key, data, 24*time.Hour) // Cache for 24 hours
}

func (g *GeoIPService) lookupExternal(ctx context.Context, ip string) (*GeoIPResult, error) {
	// This is a basic implementation. In production, you would use services like:
	// - MaxMind GeoIP2
	// - IP2Location
	// - IPapi
	// - ipinfo.io
	// - Abstract API

	// For demo purposes, using a free service (ipapi.co)
	url := fmt.Sprintf("https://ipapi.co/%s/json/", ip)
	
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var apiResult struct {
		IP          string  `json:"ip"`
		Country     string  `json:"country_name"`
		CountryCode string  `json:"country_code"`
		Region      string  `json:"region"`
		City        string  `json:"city"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		Timezone    string  `json:"timezone"`
		ISP         string  `json:"org"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResult); err != nil {
		return nil, err
	}

	result := &GeoIPResult{
		IP:          apiResult.IP,
		CountryCode: apiResult.CountryCode,
		CountryName: apiResult.Country,
		Region:      apiResult.Region,
		City:        apiResult.City,
		Latitude:    &apiResult.Latitude,
		Longitude:   &apiResult.Longitude,
		Timezone:    apiResult.Timezone,
		ISP:         apiResult.ISP,
	}

	return result, nil
}

func (g *GeoIPService) detectBrowser(ua string) (name, version string) {
	browsers := map[string]string{
		"edg/":     "Edge",
		"chrome/":  "Chrome",
		"firefox/": "Firefox",
		"safari/":  "Safari",
		"opera/":   "Opera",
		"yabrowser/": "Yandex Browser",
		"vivaldi/": "Vivaldi",
	}

	for pattern, browserName := range browsers {
		if idx := strings.Index(ua, pattern); idx != -1 {
			name = browserName
			// Try to extract version
			versionStart := idx + len(pattern)
			if versionStart < len(ua) {
				versionEnd := versionStart
				for versionEnd < len(ua) && (ua[versionEnd] >= '0' && ua[versionEnd] <= '9' || ua[versionEnd] == '.') {
					versionEnd++
				}
				if versionEnd > versionStart {
					version = ua[versionStart:versionEnd]
				}
			}
			return
		}
	}

	return "Unknown", ""
}

func (g *GeoIPService) detectOS(ua string) (name, version string) {
	osPatterns := map[string]string{
		"windows nt 10.0": "Windows 10",
		"windows nt 6.3":  "Windows 8.1",
		"windows nt 6.2":  "Windows 8",
		"windows nt 6.1":  "Windows 7",
		"windows nt":      "Windows",
		"mac os x":        "macOS",
		"macintosh":       "macOS",
		"linux":           "Linux",
		"ubuntu":          "Ubuntu",
		"android":         "Android",
		"iphone os":       "iOS",
		"cpu os":          "iOS",
		"ipad":            "iPadOS",
	}

	for pattern, osName := range osPatterns {
		if strings.Contains(ua, pattern) {
			name = osName
			
			// Try to extract version for some OSes
			if osName == "Android" {
				if idx := strings.Index(ua, "android "); idx != -1 {
					versionStart := idx + 8
					versionEnd := strings.Index(ua[versionStart:], ";")
					if versionEnd != -1 {
						version = strings.TrimSpace(ua[versionStart : versionStart+versionEnd])
					}
				}
			} else if osName == "iOS" || osName == "iPadOS" {
				if idx := strings.Index(ua, "os "); idx != -1 {
					versionStart := idx + 3
					versionEnd := strings.Index(ua[versionStart:], " ")
					if versionEnd != -1 {
						version = strings.ReplaceAll(ua[versionStart:versionStart+versionEnd], "_", ".")
					}
				}
			}
			
			return
		}
	}

	return "Unknown", ""
}

// GetTopCountriesForUser returns top countries for a user's forms
func (g *GeoIPService) GetTopCountriesForUser(ctx context.Context, userID string, startDate, endDate time.Time, limit int) ([]models.CountryStats, error) {
	query := `
		SELECT 
			country_code,
			country_name,
			SUM(total_views) as views,
			SUM(total_submissions) as submissions,
			AVG(conversion_rate) as conversion_rate
		FROM form_geographic_analytics 
		WHERE user_id = ? AND date BETWEEN ? AND ?
		GROUP BY country_code, country_name 
		ORDER BY views DESC 
		LIMIT ?
	`

	rows, err := g.db.QueryContext(ctx, query, userID, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []models.CountryStats
	for rows.Next() {
		var country models.CountryStats
		err := rows.Scan(&country.CountryCode, &country.CountryName, 
			&country.Views, &country.Submissions, &country.ConversionRate)
		if err != nil {
			continue
		}
		countries = append(countries, country)
	}

	return countries, nil
}

// GetDeviceBreakdownForUser returns device breakdown for a user's forms
func (g *GeoIPService) GetDeviceBreakdownForUser(ctx context.Context, userID string, startDate, endDate time.Time) ([]models.DeviceStats, error) {
	query := `
		SELECT 
			device_type,
			SUM(total_views) as views,
			SUM(total_submissions) as submissions,
			AVG(conversion_rate) as conversion_rate
		FROM form_device_analytics 
		WHERE user_id = ? AND date BETWEEN ? AND ?
		GROUP BY device_type 
		ORDER BY views DESC
	`

	rows, err := g.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.DeviceStats
	for rows.Next() {
		var device models.DeviceStats
		err := rows.Scan(&device.DeviceType, &device.Views, &device.Submissions, &device.ConversionRate)
		if err != nil {
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}