package services

import (
	"context"
	"encoding/json"
	"fmt"
	"formhub/internal/models"
	"formhub/pkg/database"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
)

type RealTimeService struct {
	db               *sqlx.DB
	redis            *database.RedisClient
	analyticsService *AnalyticsService
	connections      sync.Map // map[string]*WebSocketConnection
	upgrader         websocket.Upgrader
}

type WebSocketConnection struct {
	UserID     uuid.UUID
	Connection *websocket.Conn
	Send       chan []byte
	LastPing   time.Time
	Subscriptions map[string]bool // What events this connection is subscribed to
	mu         sync.RWMutex
}

type RealTimeEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    uuid.UUID              `json:"user_id,omitempty"`
	FormID    *uuid.UUID             `json:"form_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
}

type LiveMetric struct {
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Change      *float64    `json:"change,omitempty"` // Percentage change from previous period
	Trend       string      `json:"trend,omitempty"`  // up, down, stable
	Timestamp   time.Time   `json:"timestamp"`
	FormID      *uuid.UUID  `json:"form_id,omitempty"`
}

// Event types for real-time updates
const (
	EventTypeNewSubmission    = "new_submission"
	EventTypeSpamDetected     = "spam_detected"
	EventTypeFormView         = "form_view"
	EventTypeConversionUpdate = "conversion_update"
	EventTypeMetricUpdate     = "metric_update"
	EventTypeAlert           = "alert"
	EventTypeSystemHealth    = "system_health"
	EventTypeUserActivity    = "user_activity"
)

func NewRealTimeService(db *sqlx.DB, redis *database.RedisClient, analyticsService *AnalyticsService) *RealTimeService {
	return &RealTimeService{
		db:               db,
		redis:            redis,
		analyticsService: analyticsService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket handles WebSocket connections
func (r *RealTimeService) HandleWebSocket(c *gin.Context) {
	// Get user ID from context (assuming authenticated)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := r.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create WebSocket connection wrapper
	wsConn := &WebSocketConnection{
		UserID:        userUUID,
		Connection:    conn,
		Send:          make(chan []byte, 256),
		LastPing:      time.Now(),
		Subscriptions: make(map[string]bool),
	}

	// Store connection
	connectionID := fmt.Sprintf("%s_%d", userUUID.String(), time.Now().UnixNano())
	r.connections.Store(connectionID, wsConn)

	// Start goroutines for reading and writing
	go r.handleWebSocketReader(wsConn, connectionID)
	go r.handleWebSocketWriter(wsConn, connectionID)

	// Send initial data
	r.sendInitialData(wsConn)

	log.Printf("WebSocket connection established for user %s", userUUID)
}

// StartRealTimeUpdates starts the background service for real-time updates
func (r *RealTimeService) StartRealTimeUpdates(ctx context.Context) {
	// Start metrics update ticker
	metricsTicker := time.NewTicker(30 * time.Second) // Update every 30 seconds
	defer metricsTicker.Stop()

	// Start ping ticker for connection health
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Start cleanup ticker
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-metricsTicker.C:
			r.broadcastMetricsUpdate()
		case <-pingTicker.C:
			r.pingConnections()
		case <-cleanupTicker.C:
			r.cleanupConnections()
		}
	}
}

// BroadcastEvent broadcasts an event to all relevant connections
func (r *RealTimeService) BroadcastEvent(event *RealTimeEvent) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal real-time event: %v", err)
		return
	}

	r.connections.Range(func(key, value interface{}) bool {
		conn := value.(*WebSocketConnection)
		
		// Check if user should receive this event
		if r.shouldReceiveEvent(conn, event) {
			select {
			case conn.Send <- eventJSON:
			default:
				// Connection is blocked, close it
				r.closeConnection(key.(string))
			}
		}
		return true
	})
}

// BroadcastNewSubmission broadcasts a new submission event
func (r *RealTimeService) BroadcastNewSubmission(userID, formID uuid.UUID, submission *models.Submission) {
	event := &RealTimeEvent{
		Type:      EventTypeNewSubmission,
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		FormID:    &formID,
		Data: map[string]interface{}{
			"submission_id": submission.ID,
			"form_id":       submission.FormID,
			"is_spam":       submission.IsSpam,
			"ip_address":    submission.IPAddress,
			"created_at":    submission.CreatedAt,
		},
	}

	r.BroadcastEvent(event)

	// Update live metrics
	r.updateLiveMetrics(userID, formID, "new_submission")
}

// BroadcastSpamDetected broadcasts a spam detection event
func (r *RealTimeService) BroadcastSpamDetected(userID, formID uuid.UUID, submission *models.Submission) {
	event := &RealTimeEvent{
		Type:      EventTypeSpamDetected,
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		FormID:    &formID,
		Data: map[string]interface{}{
			"submission_id": submission.ID,
			"form_id":       submission.FormID,
			"spam_score":    submission.SpamScore,
			"ip_address":    submission.IPAddress,
			"created_at":    submission.CreatedAt,
		},
	}

	r.BroadcastEvent(event)
}

// BroadcastFormView broadcasts a form view event
func (r *RealTimeService) BroadcastFormView(userID, formID uuid.UUID, sessionID string) {
	event := &RealTimeEvent{
		Type:      EventTypeFormView,
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		FormID:    &formID,
		Data: map[string]interface{}{
			"form_id":    formID,
			"session_id": sessionID,
		},
	}

	r.BroadcastEvent(event)

	// Update live metrics
	r.updateLiveMetrics(userID, formID, "form_view")
}

// GetLiveMetrics returns current live metrics for a user
func (r *RealTimeService) GetLiveMetrics(ctx context.Context, userID uuid.UUID) ([]LiveMetric, error) {
	var metrics []LiveMetric

	// Get submissions in last hour
	var submissionsLastHour int
	err := r.db.GetContext(ctx, &submissionsLastHour, `
		SELECT COUNT(*) FROM submissions s
		INNER JOIN forms f ON s.form_id = f.id
		WHERE f.user_id = ? AND s.created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
	`, userID)
	if err == nil {
		metrics = append(metrics, LiveMetric{
			Name:      "submissions_last_hour",
			Value:     submissionsLastHour,
			Timestamp: time.Now().UTC(),
		})
	}

	// Get views in last hour from Redis
	viewsKey := fmt.Sprintf("live_metrics:%s:views:hour", userID)
	views, _ := r.redis.Client.Get(ctx, viewsKey).Int()
	metrics = append(metrics, LiveMetric{
		Name:      "views_last_hour",
		Value:     views,
		Timestamp: time.Now().UTC(),
	})

	// Get spam blocked in last hour
	var spamLastHour int
	err = r.db.GetContext(ctx, &spamLastHour, `
		SELECT COUNT(*) FROM submissions s
		INNER JOIN forms f ON s.form_id = f.id
		WHERE f.user_id = ? AND s.is_spam = TRUE AND s.created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
	`, userID)
	if err == nil {
		metrics = append(metrics, LiveMetric{
			Name:      "spam_blocked_last_hour",
			Value:     spamLastHour,
			Timestamp: time.Now().UTC(),
		})
	}

	// Get conversion rate
	if views > 0 && submissionsLastHour > 0 {
		conversionRate := float64(submissionsLastHour) / float64(views) * 100
		metrics = append(metrics, LiveMetric{
			Name:      "conversion_rate_last_hour",
			Value:     fmt.Sprintf("%.2f%%", conversionRate),
			Timestamp: time.Now().UTC(),
		})
	}

	// Get active sessions
	activeSessionsKey := fmt.Sprintf("analytics:active_sessions:%s", userID)
	activeSessions, _ := r.redis.Client.SCard(ctx, activeSessionsKey).Result()
	metrics = append(metrics, LiveMetric{
		Name:      "active_sessions",
		Value:     activeSessions,
		Timestamp: time.Now().UTC(),
	})

	return metrics, nil
}

// Private methods

func (r *RealTimeService) handleWebSocketReader(conn *WebSocketConnection, connectionID string) {
	defer func() {
		r.closeConnection(connectionID)
	}()

	conn.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Connection.SetPongHandler(func(string) error {
		conn.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.mu.Lock()
		conn.LastPing = time.Now()
		conn.mu.Unlock()
		return nil
	})

	for {
		_, message, err := conn.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %s: %v", conn.UserID, err)
			}
			break
		}

		// Handle incoming messages (subscriptions, etc.)
		r.handleClientMessage(conn, message)
	}
}

func (r *RealTimeService) handleWebSocketWriter(conn *WebSocketConnection, connectionID string) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		r.closeConnection(connectionID)
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error for user %s: %v", conn.UserID, err)
				return
			}

		case <-ticker.C:
			conn.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (r *RealTimeService) handleClientMessage(conn *WebSocketConnection, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to parse client message: %v", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		if events, ok := msg["events"].([]interface{}); ok {
			conn.mu.Lock()
			for _, event := range events {
				if eventStr, ok := event.(string); ok {
					conn.Subscriptions[eventStr] = true
				}
			}
			conn.mu.Unlock()
		}
	case "unsubscribe":
		if events, ok := msg["events"].([]interface{}); ok {
			conn.mu.Lock()
			for _, event := range events {
				if eventStr, ok := event.(string); ok {
					delete(conn.Subscriptions, eventStr)
				}
			}
			conn.mu.Unlock()
		}
	}
}

func (r *RealTimeService) sendInitialData(conn *WebSocketConnection) {
	// Send initial metrics
	ctx := context.Background()
	metrics, err := r.GetLiveMetrics(ctx, conn.UserID)
	if err != nil {
		return
	}

	initialData := map[string]interface{}{
		"type":    "initial_data",
		"metrics": metrics,
	}

	data, _ := json.Marshal(initialData)
	select {
	case conn.Send <- data:
	default:
		// Connection is blocked
	}
}

func (r *RealTimeService) shouldReceiveEvent(conn *WebSocketConnection, event *RealTimeEvent) bool {
	// Check if event is for this user
	if event.UserID != uuid.Nil && event.UserID != conn.UserID {
		return false
	}

	// Check subscriptions
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	
	if len(conn.Subscriptions) == 0 {
		// Default: receive all events for the user
		return event.UserID == conn.UserID
	}

	return conn.Subscriptions[event.Type]
}

func (r *RealTimeService) closeConnection(connectionID string) {
	if value, ok := r.connections.LoadAndDelete(connectionID); ok {
		conn := value.(*WebSocketConnection)
		close(conn.Send)
		conn.Connection.Close()
		log.Printf("Closed WebSocket connection for user %s", conn.UserID)
	}
}

func (r *RealTimeService) broadcastMetricsUpdate() {
	// Get all unique user IDs from connections
	userIDs := make(map[uuid.UUID]bool)
	r.connections.Range(func(key, value interface{}) bool {
		conn := value.(*WebSocketConnection)
		userIDs[conn.UserID] = true
		return true
	})

	// Update metrics for each user
	for userID := range userIDs {
		go func(uid uuid.UUID) {
			ctx := context.Background()
			metrics, err := r.GetLiveMetrics(ctx, uid)
			if err != nil {
				return
			}

			event := &RealTimeEvent{
				Type:      EventTypeMetricUpdate,
				Timestamp: time.Now().UTC(),
				UserID:    uid,
				Data: map[string]interface{}{
					"metrics": metrics,
				},
			}

			r.BroadcastEvent(event)
		}(userID)
	}
}

func (r *RealTimeService) pingConnections() {
	r.connections.Range(func(key, value interface{}) bool {
		conn := value.(*WebSocketConnection)
		
		conn.mu.RLock()
		lastPing := conn.LastPing
		conn.mu.RUnlock()

		// Close connections that haven't responded to ping in 2 minutes
		if time.Since(lastPing) > 2*time.Minute {
			r.closeConnection(key.(string))
		}
		return true
	})
}

func (r *RealTimeService) cleanupConnections() {
	r.connections.Range(func(key, value interface{}) bool {
		conn := value.(*WebSocketConnection)
		
		// Check if connection is still alive
		select {
		case conn.Send <- []byte(`{"type":"ping"}`):
		default:
			// Connection is blocked, close it
			r.closeConnection(key.(string))
		}
		return true
	})
}

func (r *RealTimeService) updateLiveMetrics(userID, formID uuid.UUID, eventType string) {
	ctx := context.Background()

	switch eventType {
	case "form_view":
		// Increment view count in Redis
		viewsKey := fmt.Sprintf("live_metrics:%s:views:hour", userID)
		r.redis.Client.Incr(ctx, viewsKey)
		r.redis.Client.Expire(ctx, viewsKey, time.Hour)

		// Form-specific views
		formViewsKey := fmt.Sprintf("live_metrics:%s:form:%s:views:hour", userID, formID)
		r.redis.Client.Incr(ctx, formViewsKey)
		r.redis.Client.Expire(ctx, formViewsKey, time.Hour)

	case "new_submission":
		// Update submission count
		submissionsKey := fmt.Sprintf("live_metrics:%s:submissions:hour", userID)
		r.redis.Client.Incr(ctx, submissionsKey)
		r.redis.Client.Expire(ctx, submissionsKey, time.Hour)

		// Form-specific submissions
		formSubmissionsKey := fmt.Sprintf("live_metrics:%s:form:%s:submissions:hour", userID, formID)
		r.redis.Client.Incr(ctx, formSubmissionsKey)
		r.redis.Client.Expire(ctx, formSubmissionsKey, time.Hour)
	}
}

// GetActiveConnections returns the number of active WebSocket connections
func (r *RealTimeService) GetActiveConnections() int {
	count := 0
	r.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetActiveConnectionsForUser returns the number of active connections for a specific user
func (r *RealTimeService) GetActiveConnectionsForUser(userID uuid.UUID) int {
	count := 0
	r.connections.Range(func(key, value interface{}) bool {
		conn := value.(*WebSocketConnection)
		if conn.UserID == userID {
			count++
		}
		return true
	})
	return count
}

// BroadcastAlert broadcasts an alert to users
func (r *RealTimeService) BroadcastAlert(userID uuid.UUID, alertType string, message string, severity string) {
	event := &RealTimeEvent{
		Type:      EventTypeAlert,
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		Data: map[string]interface{}{
			"alert_type": alertType,
			"message":    message,
			"severity":   severity,
		},
	}

	r.BroadcastEvent(event)
}

// BroadcastSystemHealth broadcasts system health updates
func (r *RealTimeService) BroadcastSystemHealth(health map[string]interface{}) {
	event := &RealTimeEvent{
		Type:      EventTypeSystemHealth,
		Timestamp: time.Now().UTC(),
		Data:      health,
	}

	// Broadcast to all connections (no specific user)
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}

	r.connections.Range(func(key, value interface{}) bool {
		conn := value.(*WebSocketConnection)
		select {
		case conn.Send <- eventJSON:
		default:
			r.closeConnection(key.(string))
		}
		return true
	})
}