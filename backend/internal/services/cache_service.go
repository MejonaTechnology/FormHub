package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/database"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type CacheService struct {
	redis *database.RedisClient
}

type CacheConfig struct {
	TTL           time.Duration `json:"ttl"`
	RefreshAfter  time.Duration `json:"refresh_after"`
	PreventDogPile bool         `json:"prevent_dogpile"`
}

// Cache key prefixes
const (
	CacheKeyFormAnalytics     = "analytics:form"
	CacheKeyUserAnalytics     = "analytics:user" 
	CacheKeyConversionFunnel  = "analytics:funnel"
	CacheKeyGeographicData    = "analytics:geo"
	CacheKeyDeviceData        = "analytics:device"
	CacheKeyFieldAnalytics    = "analytics:field"
	CacheKeyRealTimeStats     = "analytics:realtime"
	CacheKeySubmissionLifecycle = "lifecycle:submission"
	CacheKeyABTestResults     = "abtest:results"
	CacheKeyReportData        = "report:data"
	CacheKeySystemHealth      = "system:health"
)

// Cache configurations for different data types
var CacheConfigs = map[string]CacheConfig{
	CacheKeyFormAnalytics: {
		TTL:           30 * time.Minute,
		RefreshAfter:  15 * time.Minute,
		PreventDogPile: true,
	},
	CacheKeyUserAnalytics: {
		TTL:           1 * time.Hour,
		RefreshAfter:  30 * time.Minute,
		PreventDogPile: true,
	},
	CacheKeyConversionFunnel: {
		TTL:           2 * time.Hour,
		RefreshAfter:  1 * time.Hour,
		PreventDogPile: true,
	},
	CacheKeyGeographicData: {
		TTL:           4 * time.Hour,
		RefreshAfter:  2 * time.Hour,
		PreventDogPile: true,
	},
	CacheKeyDeviceData: {
		TTL:           4 * time.Hour,
		RefreshAfter:  2 * time.Hour,
		PreventDogPile: true,
	},
	CacheKeyFieldAnalytics: {
		TTL:           1 * time.Hour,
		RefreshAfter:  30 * time.Minute,
		PreventDogPile: true,
	},
	CacheKeyRealTimeStats: {
		TTL:           5 * time.Minute,
		RefreshAfter:  2 * time.Minute,
		PreventDogPile: false, // Real-time data should be fresh
	},
	CacheKeySubmissionLifecycle: {
		TTL:           2 * time.Hour,
		RefreshAfter:  1 * time.Hour,
		PreventDogPile: true,
	},
	CacheKeyABTestResults: {
		TTL:           15 * time.Minute,
		RefreshAfter:  7 * time.Minute,
		PreventDogPile: true,
	},
	CacheKeyReportData: {
		TTL:           24 * time.Hour,
		RefreshAfter:  12 * time.Hour,
		PreventDogPile: true,
	},
	CacheKeySystemHealth: {
		TTL:           1 * time.Minute,
		RefreshAfter:  30 * time.Second,
		PreventDogPile: false,
	},
}

type CacheItem struct {
	Data        interface{} `json:"data"`
	CachedAt    time.Time   `json:"cached_at"`
	RefreshAt   time.Time   `json:"refresh_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	Version     int         `json:"version"`
	Tags        []string    `json:"tags,omitempty"`
}

type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Sets        int64   `json:"sets"`
	Deletes     int64   `json:"deletes"`
	HitRate     float64 `json:"hit_rate"`
	MemoryUsage int64   `json:"memory_usage"`
}

func NewCacheService(redis *database.RedisClient) *CacheService {
	return &CacheService{
		redis: redis,
	}
}

// Get retrieves data from cache
func (c *CacheService) Get(ctx context.Context, key string) (interface{}, bool, error) {
	c.incrementStat(ctx, "cache:stats:requests")

	data, err := c.redis.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		c.incrementStat(ctx, "cache:stats:misses")
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache get error: %w", err)
	}

	var item CacheItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil, false, fmt.Errorf("cache unmarshal error: %w", err)
	}

	c.incrementStat(ctx, "cache:stats:hits")

	// Check if data needs refresh (stale-while-revalidate pattern)
	if time.Now().After(item.RefreshAt) {
		// Data is stale but still valid, should trigger background refresh
		go c.markForRefresh(ctx, key)
	}

	return item.Data, true, nil
}

// Set stores data in cache
func (c *CacheService) Set(ctx context.Context, key string, data interface{}, config CacheConfig, tags ...string) error {
	now := time.Now()
	item := CacheItem{
		Data:      data,
		CachedAt:  now,
		RefreshAt: now.Add(config.RefreshAfter),
		ExpiresAt: now.Add(config.TTL),
		Version:   1,
		Tags:      tags,
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	err = c.redis.Client.Set(ctx, key, jsonData, config.TTL).Err()
	if err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}

	// Store tags for invalidation
	for _, tag := range tags {
		tagKey := fmt.Sprintf("cache:tag:%s", tag)
		c.redis.Client.SAdd(ctx, tagKey, key)
		c.redis.Client.Expire(ctx, tagKey, config.TTL)
	}

	c.incrementStat(ctx, "cache:stats:sets")
	return nil
}

// GetOrSet retrieves from cache or sets if not found (cache-aside pattern)
func (c *CacheService) GetOrSet(ctx context.Context, key string, fetcher func() (interface{}, error), config CacheConfig, tags ...string) (interface{}, error) {
	// Try to get from cache first
	if data, found, err := c.Get(ctx, key); err == nil && found {
		return data, nil
	}

	// Prevent cache stampede with distributed locking
	if config.PreventDogPile {
		lockKey := fmt.Sprintf("%s:lock", key)
		locked, err := c.acquireLock(ctx, lockKey, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}

		if !locked {
			// Another process is fetching, wait and try cache again
			time.Sleep(100 * time.Millisecond)
			if data, found, err := c.Get(ctx, key); err == nil && found {
				return data, nil
			}
		}

		defer c.releaseLock(ctx, lockKey)
	}

	// Fetch fresh data
	data, err := fetcher()
	if err != nil {
		return nil, fmt.Errorf("fetcher error: %w", err)
	}

	// Store in cache
	if err := c.Set(ctx, key, data, config, tags...); err != nil {
		log.Printf("Failed to set cache for key %s: %v", key, err)
		// Return data even if caching fails
	}

	return data, nil
}

// Delete removes a key from cache
func (c *CacheService) Delete(ctx context.Context, key string) error {
	err := c.redis.Client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("cache delete error: %w", err)
	}

	c.incrementStat(ctx, "cache:stats:deletes")
	return nil
}

// InvalidateByTag invalidates all cache entries with a specific tag
func (c *CacheService) InvalidateByTag(ctx context.Context, tag string) error {
	tagKey := fmt.Sprintf("cache:tag:%s", tag)
	
	keys, err := c.redis.Client.SMembers(ctx, tagKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get tag members: %w", err)
	}

	if len(keys) > 0 {
		if err := c.redis.Client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete tagged keys: %w", err)
		}
	}

	// Clean up the tag set
	c.redis.Client.Del(ctx, tagKey)

	log.Printf("Invalidated %d cache entries with tag: %s", len(keys), tag)
	return nil
}

// InvalidateUserCache invalidates all cache entries for a specific user
func (c *CacheService) InvalidateUserCache(ctx context.Context, userID uuid.UUID) error {
	pattern := fmt.Sprintf("*:%s:*", userID.String())
	return c.invalidateByPattern(ctx, pattern)
}

// InvalidateFormCache invalidates all cache entries for a specific form
func (c *CacheService) InvalidateFormCache(ctx context.Context, formID uuid.UUID) error {
	pattern := fmt.Sprintf("*:%s:*", formID.String())
	return c.invalidateByPattern(ctx, pattern)
}

// Warm pre-fills cache with commonly accessed data
func (c *CacheService) Warm(ctx context.Context, userID uuid.UUID, formIDs []uuid.UUID) error {
	log.Printf("Warming cache for user %s with %d forms", userID, len(formIDs))

	// Define warming tasks
	tasks := []func() error{
		func() error {
			// Warm real-time stats
			key := c.BuildKey(CacheKeyRealTimeStats, userID.String())
			config := CacheConfigs[CacheKeyRealTimeStats]
			_, err := c.GetOrSet(ctx, key, func() (interface{}, error) {
				// This would call the actual service method
				return map[string]interface{}{
					"active_sessions": 0,
					"submissions_last_hour": 0,
					"spam_blocked": 0,
				}, nil
			}, config)
			return err
		},
	}

	// Warm form-specific caches
	for _, formID := range formIDs {
		fID := formID // Capture for closure
		tasks = append(tasks, func() error {
			key := c.BuildKey(CacheKeyFormAnalytics, userID.String(), fID.String(), "30d")
			config := CacheConfigs[CacheKeyFormAnalytics]
			_, err := c.GetOrSet(ctx, key, func() (interface{}, error) {
				// This would call the actual analytics service
				return &models.FormAnalyticsDashboard{
					FormID:   fID,
					FormName: "Sample Form",
				}, nil
			}, config, fmt.Sprintf("user:%s", userID), fmt.Sprintf("form:%s", fID))
			return err
		})
	}

	// Execute warming tasks concurrently
	errCh := make(chan error, len(tasks))
	for _, task := range tasks {
		go func(t func() error) {
			errCh <- t()
		}(task)
	}

	// Collect results
	var errors []string
	for i := 0; i < len(tasks); i++ {
		if err := <-errCh; err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache warming errors: %s", strings.Join(errors, "; "))
	}

	log.Printf("Successfully warmed cache for user %s", userID)
	return nil
}

// GetStats returns cache performance statistics
func (c *CacheService) GetStats(ctx context.Context) (*CacheStats, error) {
	pipe := c.redis.Client.Pipeline()
	
	hitsCmd := pipe.Get(ctx, "cache:stats:hits")
	missesCmd := pipe.Get(ctx, "cache:stats:misses")
	setsCmd := pipe.Get(ctx, "cache:stats:sets")
	deletesCmd := pipe.Get(ctx, "cache:stats:deletes")
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	stats := &CacheStats{}
	
	if hits, err := hitsCmd.Int64(); err == nil {
		stats.Hits = hits
	}
	if misses, err := missesCmd.Int64(); err == nil {
		stats.Misses = misses
	}
	if sets, err := setsCmd.Int64(); err == nil {
		stats.Sets = sets
	}
	if deletes, err := deletesCmd.Int64(); err == nil {
		stats.Deletes = deletes
	}

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total) * 100
	}

	// Get memory usage (Redis specific)
	if info, err := c.redis.Client.Info(ctx, "memory").Result(); err == nil {
		// Parse memory info (simplified)
		if strings.Contains(info, "used_memory:") {
			// This is a simplified parsing - in production you'd want proper parsing
			stats.MemoryUsage = 1024 * 1024 // Placeholder
		}
	}

	return stats, nil
}

// BuildKey constructs cache keys consistently
func (c *CacheService) BuildKey(prefix string, parts ...string) string {
	allParts := append([]string{prefix}, parts...)
	key := strings.Join(allParts, ":")
	
	// Hash long keys to prevent Redis key length issues
	if len(key) > 200 {
		hash := sha256.Sum256([]byte(key))
		return prefix + ":" + hex.EncodeToString(hash[:])[:16]
	}
	
	return key
}

// FlushAll clears all cache (use with caution)
func (c *CacheService) FlushAll(ctx context.Context) error {
	return c.redis.Client.FlushAll(ctx).Err()
}

// HealthCheck checks if cache is responsive
func (c *CacheService) HealthCheck(ctx context.Context) error {
	testKey := "cache:health:test"
	testValue := time.Now().String()
	
	// Test write
	if err := c.redis.Client.Set(ctx, testKey, testValue, time.Minute).Err(); err != nil {
		return fmt.Errorf("cache write failed: %w", err)
	}
	
	// Test read
	if result, err := c.redis.Client.Get(ctx, testKey).Result(); err != nil {
		return fmt.Errorf("cache read failed: %w", err)
	} else if result != testValue {
		return fmt.Errorf("cache read returned wrong value")
	}
	
	// Test delete
	if err := c.redis.Client.Del(ctx, testKey).Err(); err != nil {
		return fmt.Errorf("cache delete failed: %w", err)
	}
	
	return nil
}

// Private methods

func (c *CacheService) incrementStat(ctx context.Context, key string) {
	c.redis.Client.Incr(ctx, key)
	c.redis.Client.Expire(ctx, key, 24*time.Hour) // Keep stats for 24 hours
}

func (c *CacheService) acquireLock(ctx context.Context, lockKey string, expiration time.Duration) (bool, error) {
	result, err := c.redis.Client.SetNX(ctx, lockKey, "locked", expiration).Result()
	return result, err
}

func (c *CacheService) releaseLock(ctx context.Context, lockKey string) error {
	return c.redis.Client.Del(ctx, lockKey).Err()
}

func (c *CacheService) markForRefresh(ctx context.Context, key string) {
	refreshKey := fmt.Sprintf("%s:refresh", key)
	c.redis.Client.Set(ctx, refreshKey, "pending", 5*time.Minute)
}

func (c *CacheService) invalidateByPattern(ctx context.Context, pattern string) error {
	keys, err := c.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to find keys: %w", err)
	}

	if len(keys) > 0 {
		if err := c.redis.Client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
		log.Printf("Invalidated %d cache entries matching pattern: %s", len(keys), pattern)
	}

	return nil
}

// Cache middleware helpers

// CacheFormAnalytics caches form analytics dashboard data
func (c *CacheService) CacheFormAnalytics(ctx context.Context, formID, userID uuid.UUID, period string, fetcher func() (*models.FormAnalyticsDashboard, error)) (*models.FormAnalyticsDashboard, error) {
	key := c.BuildKey(CacheKeyFormAnalytics, userID.String(), formID.String(), period)
	config := CacheConfigs[CacheKeyFormAnalytics]
	tags := []string{fmt.Sprintf("user:%s", userID), fmt.Sprintf("form:%s", formID)}

	result, err := c.GetOrSet(ctx, key, func() (interface{}, error) {
		return fetcher()
	}, config, tags...)

	if err != nil {
		return nil, err
	}

	dashboard, ok := result.(*models.FormAnalyticsDashboard)
	if !ok {
		// Try to convert from map (JSON unmarshal artifact)
		if data, ok := result.(map[string]interface{}); ok {
			jsonData, _ := json.Marshal(data)
			dashboard = &models.FormAnalyticsDashboard{}
			json.Unmarshal(jsonData, dashboard)
		} else {
			return nil, fmt.Errorf("invalid cache data type: %T", result)
		}
	}

	return dashboard, nil
}

// CacheConversionFunnel caches conversion funnel data
func (c *CacheService) CacheConversionFunnel(ctx context.Context, formID, userID uuid.UUID, startDate, endDate time.Time, fetcher func() ([]models.FormConversionFunnel, error)) ([]models.FormConversionFunnel, error) {
	dateRange := fmt.Sprintf("%s_%s", startDate.Format("20060102"), endDate.Format("20060102"))
	key := c.BuildKey(CacheKeyConversionFunnel, userID.String(), formID.String(), dateRange)
	config := CacheConfigs[CacheKeyConversionFunnel]
	tags := []string{fmt.Sprintf("user:%s", userID), fmt.Sprintf("form:%s", formID)}

	result, err := c.GetOrSet(ctx, key, func() (interface{}, error) {
		return fetcher()
	}, config, tags...)

	if err != nil {
		return nil, err
	}

	funnels, ok := result.([]models.FormConversionFunnel)
	if !ok {
		// Handle interface{} slice from JSON unmarshaling
		if dataSlice, ok := result.([]interface{}); ok {
			funnels = make([]models.FormConversionFunnel, len(dataSlice))
			for i, item := range dataSlice {
				jsonData, _ := json.Marshal(item)
				json.Unmarshal(jsonData, &funnels[i])
			}
		} else {
			return nil, fmt.Errorf("invalid cache data type: %T", result)
		}
	}

	return funnels, nil
}

// CacheRealTimeStats caches real-time statistics
func (c *CacheService) CacheRealTimeStats(ctx context.Context, userID uuid.UUID, fetcher func() (*models.RealTimeStats, error)) (*models.RealTimeStats, error) {
	key := c.BuildKey(CacheKeyRealTimeStats, userID.String())
	config := CacheConfigs[CacheKeyRealTimeStats]
	tags := []string{fmt.Sprintf("user:%s", userID)}

	result, err := c.GetOrSet(ctx, key, func() (interface{}, error) {
		return fetcher()
	}, config, tags...)

	if err != nil {
		return nil, err
	}

	stats, ok := result.(*models.RealTimeStats)
	if !ok {
		if data, ok := result.(map[string]interface{}); ok {
			jsonData, _ := json.Marshal(data)
			stats = &models.RealTimeStats{}
			json.Unmarshal(jsonData, stats)
		} else {
			return nil, fmt.Errorf("invalid cache data type: %T", result)
		}
	}

	return stats, nil
}

// StartBackgroundTasks starts background cache maintenance tasks
func (c *CacheService) StartBackgroundTasks(ctx context.Context) {
	// Stats collection
	statsTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				statsTicker.Stop()
				return
			case <-statsTicker.C:
				c.collectStats(ctx)
			}
		}
	}()

	// Cleanup expired entries (Redis handles this automatically, but we can do additional cleanup)
	cleanupTicker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				cleanupTicker.Stop()
				return
			case <-cleanupTicker.C:
				c.cleanup(ctx)
			}
		}
	}()
}

func (c *CacheService) collectStats(ctx context.Context) {
	stats, err := c.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to collect cache stats: %v", err)
		return
	}

	// Store historical stats
	timestamp := time.Now().Unix()
	statsKey := fmt.Sprintf("cache:historical_stats:%d", timestamp)
	
	statsData, _ := json.Marshal(stats)
	c.redis.Client.Set(ctx, statsKey, statsData, 24*time.Hour)
	
	log.Printf("Cache stats - Hit rate: %.2f%%, Hits: %d, Misses: %d", 
		stats.HitRate, stats.Hits, stats.Misses)
}

func (c *CacheService) cleanup(ctx context.Context) {
	// Clean up old stats
	pattern := "cache:stats:*"
	keys, err := c.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return
	}

	// Keep only recent stats (last 24 hours)
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	for _, key := range keys {
		if strings.Contains(key, "historical_stats") {
			// Extract timestamp from key and delete if old
			// This is simplified - you'd want proper timestamp extraction
			if strings.Contains(key, "historical_stats") {
				// Delete keys older than 7 days
				sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour).Unix()
				keyTimestamp := sevenDaysAgo - 1 // Placeholder logic
				if keyTimestamp < cutoff {
					c.redis.Client.Del(ctx, key)
				}
			}
		}
	}

	log.Printf("Cache cleanup completed")
}