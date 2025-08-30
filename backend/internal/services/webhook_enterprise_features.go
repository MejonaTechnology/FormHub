package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Circuit Breaker Implementation

// IsOpen checks if the circuit breaker is open for an endpoint
func (cb *CircuitBreaker) IsOpen(endpointID string) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	state, exists := cb.endpoints[endpointID]
	if !exists {
		return false // No state means circuit is closed
	}
	
	now := time.Now()
	
	switch state.State {
	case "open":
		// Check if it's time to try half-open
		if now.After(state.NextReset) {
			cb.mu.RUnlock()
			cb.mu.Lock()
			if state.State == "open" && now.After(state.NextReset) {
				state.State = "half_open"
				log.Printf("Circuit breaker for endpoint %s moved to half-open", endpointID)
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return false // Allow one request in half-open state
		}
		return true
	case "half_open":
		return false // Allow requests in half-open state
	default: // closed
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess(endpointID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	state, exists := cb.endpoints[endpointID]
	if !exists {
		state = &EndpointState{
			State:    "closed",
			Failures: 0,
		}
		cb.endpoints[endpointID] = state
	}
	
	if state.State == "half_open" {
		// Success in half-open state, close the circuit
		state.State = "closed"
		state.Failures = 0
		log.Printf("Circuit breaker for endpoint %s closed after successful request", endpointID)
	} else if state.State == "closed" {
		// Reset failure count on success
		state.Failures = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure(endpointID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	state, exists := cb.endpoints[endpointID]
	if !exists {
		state = &EndpointState{
			State:    "closed",
			Failures: 0,
		}
		cb.endpoints[endpointID] = state
	}
	
	state.Failures++
	state.LastFailure = time.Now()
	
	if state.State == "half_open" {
		// Failure in half-open state, go back to open
		state.State = "open"
		state.NextReset = time.Now().Add(cb.resetTimeout)
		log.Printf("Circuit breaker for endpoint %s opened after failed request in half-open state", endpointID)
	} else if state.State == "closed" && state.Failures >= cb.maxFailures {
		// Too many failures, open the circuit
		state.State = "open"
		state.NextReset = time.Now().Add(cb.resetTimeout)
		log.Printf("Circuit breaker for endpoint %s opened after %d failures", endpointID, state.Failures)
	}
}

// GetState returns the current state of the circuit breaker for an endpoint
func (cb *CircuitBreaker) GetState(endpointID string) string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	if state, exists := cb.endpoints[endpointID]; exists {
		return state.State
	}
	return "closed"
}

// Reset resets the circuit breaker for an endpoint
func (cb *CircuitBreaker) Reset(endpointID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	state, exists := cb.endpoints[endpointID]
	if !exists {
		return
	}
	
	state.State = "closed"
	state.Failures = 0
	log.Printf("Circuit breaker for endpoint %s manually reset", endpointID)
}

// Load Balancer Implementation

// SelectEndpoint selects an endpoint based on the configured strategy
func (wlb *WebhookLoadBalancer) SelectEndpoint(endpoints []WebhookEndpoint) *WebhookEndpoint {
	wlb.mu.Lock()
	defer wlb.mu.Unlock()
	
	if len(endpoints) == 0 {
		return nil
	}
	
	// Convert to weighted endpoints if needed
	wlb.updateWeightedEndpoints(endpoints)
	
	switch wlb.strategy {
	case "round_robin":
		return wlb.selectRoundRobin()
	case "weighted":
		return wlb.selectWeighted()
	case "priority":
		return wlb.selectPriority()
	case "random":
		return wlb.selectRandom()
	case "health_based":
		return wlb.selectHealthBased()
	default:
		return wlb.selectRoundRobin()
	}
}

// updateWeightedEndpoints updates the internal weighted endpoints list
func (wlb *WebhookLoadBalancer) updateWeightedEndpoints(endpoints []WebhookEndpoint) {
	// Only update if endpoints have changed
	if len(wlb.endpoints) != len(endpoints) {
		wlb.endpoints = make([]WeightedEndpoint, len(endpoints))
		for i, endpoint := range endpoints {
			wlb.endpoints[i] = WeightedEndpoint{
				Endpoint:    &endpoints[i],
				Weight:      wlb.getEndpointWeight(&endpoints[i]),
				HealthScore: 1.0, // Default healthy score
				LastCheck:   time.Now(),
			}
		}
	}
}

// selectRoundRobin selects endpoint using round-robin strategy
func (wlb *WebhookLoadBalancer) selectRoundRobin() *WebhookEndpoint {
	if len(wlb.endpoints) == 0 {
		return nil
	}
	
	// Find next healthy endpoint
	start := wlb.currentIdx
	for i := 0; i < len(wlb.endpoints); i++ {
		idx := (start + i) % len(wlb.endpoints)
		endpoint := &wlb.endpoints[idx]
		
		if endpoint.Endpoint.Enabled && endpoint.HealthScore > 0.5 {
			wlb.currentIdx = (idx + 1) % len(wlb.endpoints)
			return endpoint.Endpoint
		}
	}
	
	// If no healthy endpoint found, return first enabled one
	for _, endpoint := range wlb.endpoints {
		if endpoint.Endpoint.Enabled {
			return endpoint.Endpoint
		}
	}
	
	return nil
}

// selectWeighted selects endpoint using weighted strategy
func (wlb *WebhookLoadBalancer) selectWeighted() *WebhookEndpoint {
	totalWeight := 0
	for _, endpoint := range wlb.endpoints {
		if endpoint.Endpoint.Enabled && endpoint.HealthScore > 0.5 {
			totalWeight += endpoint.Weight
		}
	}
	
	if totalWeight == 0 {
		return wlb.selectRoundRobin() // Fallback
	}
	
	// Generate random number and find corresponding endpoint
	target := rand.Intn(totalWeight)
	current := 0
	
	for _, endpoint := range wlb.endpoints {
		if endpoint.Endpoint.Enabled && endpoint.HealthScore > 0.5 {
			current += endpoint.Weight
			if current > target {
				return endpoint.Endpoint
			}
		}
	}
	
	return wlb.selectRoundRobin() // Fallback
}

// selectPriority selects endpoint based on priority (lowest number = highest priority)
func (wlb *WebhookLoadBalancer) selectPriority() *WebhookEndpoint {
	var bestEndpoint *WebhookEndpoint
	bestPriority := math.MaxInt32
	
	for _, endpoint := range wlb.endpoints {
		if endpoint.Endpoint.Enabled && endpoint.HealthScore > 0.5 {
			if endpoint.Endpoint.Priority < bestPriority {
				bestPriority = endpoint.Endpoint.Priority
				bestEndpoint = endpoint.Endpoint
			}
		}
	}
	
	if bestEndpoint == nil {
		return wlb.selectRoundRobin() // Fallback
	}
	
	return bestEndpoint
}

// selectRandom selects a random healthy endpoint
func (wlb *WebhookLoadBalancer) selectRandom() *WebhookEndpoint {
	var healthyEndpoints []*WebhookEndpoint
	
	for _, endpoint := range wlb.endpoints {
		if endpoint.Endpoint.Enabled && endpoint.HealthScore > 0.5 {
			healthyEndpoints = append(healthyEndpoints, endpoint.Endpoint)
		}
	}
	
	if len(healthyEndpoints) == 0 {
		return wlb.selectRoundRobin() // Fallback
	}
	
	return healthyEndpoints[rand.Intn(len(healthyEndpoints))]
}

// selectHealthBased selects endpoint based on health score
func (wlb *WebhookLoadBalancer) selectHealthBased() *WebhookEndpoint {
	var bestEndpoint *WebhookEndpoint
	bestScore := 0.0
	
	for _, endpoint := range wlb.endpoints {
		if endpoint.Endpoint.Enabled && endpoint.HealthScore > bestScore {
			bestScore = endpoint.HealthScore
			bestEndpoint = endpoint.Endpoint
		}
	}
	
	if bestEndpoint == nil {
		return wlb.selectRoundRobin() // Fallback
	}
	
	return bestEndpoint
}

// getEndpointWeight calculates weight for an endpoint
func (wlb *WebhookLoadBalancer) getEndpointWeight(endpoint *WebhookEndpoint) int {
	// Base weight on priority (lower priority = higher weight)
	baseWeight := 10 - endpoint.Priority
	if baseWeight < 1 {
		baseWeight = 1
	}
	
	// You could add more sophisticated weight calculation here
	// based on historical performance, response times, etc.
	
	return baseWeight
}

// UpdateEndpointHealth updates the health score for an endpoint
func (wlb *WebhookLoadBalancer) UpdateEndpointHealth(endpointID string, healthScore float64) {
	wlb.mu.Lock()
	defer wlb.mu.Unlock()
	
	for i := range wlb.endpoints {
		if wlb.endpoints[i].Endpoint.ID == endpointID {
			wlb.endpoints[i].HealthScore = healthScore
			wlb.endpoints[i].LastCheck = time.Now()
			break
		}
	}
}

// SetStrategy sets the load balancing strategy
func (wlb *WebhookLoadBalancer) SetStrategy(strategy string) {
	wlb.mu.Lock()
	defer wlb.mu.Unlock()
	
	validStrategies := map[string]bool{
		"round_robin":  true,
		"weighted":     true,
		"priority":     true,
		"random":       true,
		"health_based": true,
	}
	
	if validStrategies[strategy] {
		wlb.strategy = strategy
		log.Printf("Load balancer strategy changed to: %s", strategy)
	} else {
		log.Printf("Invalid load balancer strategy: %s", strategy)
	}
}

// Enterprise Security Features

// WebhookFirewall provides IP whitelisting and blacklisting
type WebhookFirewall struct {
	allowedIPs   map[string]bool
	blockedIPs   map[string]bool
	allowedCIDRs []*net.IPNet
	blockedCIDRs []*net.IPNet
	mu           sync.RWMutex
}

// NewWebhookFirewall creates a new webhook firewall
func NewWebhookFirewall() *WebhookFirewall {
	return &WebhookFirewall{
		allowedIPs:   make(map[string]bool),
		blockedIPs:   make(map[string]bool),
		allowedCIDRs: make([]*net.IPNet, 0),
		blockedCIDRs: make([]*net.IPNet, 0),
	}
}

// IsAllowed checks if an IP address is allowed
func (wf *WebhookFirewall) IsAllowed(ip string) bool {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	
	// Check if explicitly blocked
	if wf.blockedIPs[ip] {
		return false
	}
	
	// Check blocked CIDRs
	ipAddr := net.ParseIP(ip)
	if ipAddr != nil {
		for _, cidr := range wf.blockedCIDRs {
			if cidr.Contains(ipAddr) {
				return false
			}
		}
	}
	
	// If there are allowed IPs configured, check them
	if len(wf.allowedIPs) > 0 || len(wf.allowedCIDRs) > 0 {
		// Check explicitly allowed IPs
		if wf.allowedIPs[ip] {
			return true
		}
		
		// Check allowed CIDRs
		if ipAddr != nil {
			for _, cidr := range wf.allowedCIDRs {
				if cidr.Contains(ipAddr) {
					return true
				}
			}
		}
		
		// Not in allowed list
		return false
	}
	
	// No restrictions configured, allow all
	return true
}

// AddAllowedIP adds an IP to the allowed list
func (wf *WebhookFirewall) AddAllowedIP(ip string) error {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	wf.allowedIPs[ip] = true
	return nil
}

// AddBlockedIP adds an IP to the blocked list
func (wf *WebhookFirewall) AddBlockedIP(ip string) error {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	wf.blockedIPs[ip] = true
	return nil
}

// AddAllowedCIDR adds a CIDR range to the allowed list
func (wf *WebhookFirewall) AddAllowedCIDR(cidr string) error {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %s", cidr)
	}
	
	wf.allowedCIDRs = append(wf.allowedCIDRs, network)
	return nil
}

// AddBlockedCIDR adds a CIDR range to the blocked list
func (wf *WebhookFirewall) AddBlockedCIDR(cidr string) error {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %s", cidr)
	}
	
	wf.blockedCIDRs = append(wf.blockedCIDRs, network)
	return nil
}

// SSL Certificate Pinning
type CertificatePinner struct {
	pinnedCerts map[string][]string // domain -> list of certificate fingerprints
	mu          sync.RWMutex
}

// NewCertificatePinner creates a new certificate pinner
func NewCertificatePinner() *CertificatePinner {
	return &CertificatePinner{
		pinnedCerts: make(map[string][]string),
	}
}

// PinCertificate pins a certificate for a domain
func (cp *CertificatePinner) PinCertificate(domain, fingerprint string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	if cp.pinnedCerts[domain] == nil {
		cp.pinnedCerts[domain] = make([]string, 0)
	}
	
	cp.pinnedCerts[domain] = append(cp.pinnedCerts[domain], fingerprint)
}

// VerifyCertificate verifies if a certificate matches the pinned certificates
func (cp *CertificatePinner) VerifyCertificate(domain string, cert *tls.Certificate) bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	
	pinnedFingerprints, exists := cp.pinnedCerts[domain]
	if !exists || len(pinnedFingerprints) == 0 {
		return true // No pinning configured, allow
	}
	
	// Calculate certificate fingerprint (simplified implementation)
	// In production, use proper certificate fingerprint calculation
	if len(cert.Certificate) == 0 {
		return false
	}
	
	// For this example, we'll just check if the certificate exists
	// In a real implementation, you'd calculate SHA256 fingerprint
	certFingerprint := fmt.Sprintf("%x", cert.Certificate[0][:20]) // Simplified
	
	for _, pinnedFingerprint := range pinnedFingerprints {
		if certFingerprint == pinnedFingerprint {
			return true
		}
	}
	
	return false
}

// Webhook Proxy Support
type WebhookProxy struct {
	proxyURL     string
	proxyAuth    *ProxyAuth
	client       *http.Client
	mu           sync.RWMutex
}

type ProxyAuth struct {
	Username string
	Password string
	Token    string // For token-based auth
}

// NewWebhookProxy creates a new webhook proxy
func NewWebhookProxy(proxyURL string, auth *ProxyAuth) (*WebhookProxy, error) {
	proxy, err := http.NewRequest("GET", proxyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}
	
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy.URL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	
	return &WebhookProxy{
		proxyURL:  proxyURL,
		proxyAuth: auth,
		client:    client,
	}, nil
}

// GetClient returns the HTTP client configured with proxy
func (wp *WebhookProxy) GetClient() *http.Client {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.client
}

// UpdateProxy updates the proxy configuration
func (wp *WebhookProxy) UpdateProxy(proxyURL string, auth *ProxyAuth) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	proxy, err := http.NewRequest("GET", proxyURL, nil)
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %w", err)
	}
	
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy.URL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}
	
	wp.client = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	
	wp.proxyURL = proxyURL
	wp.proxyAuth = auth
	
	return nil
}

// Webhook Rate Limiter with Redis
type EnhancedRateLimiter struct {
	redis       *redis.Client
	defaultRate int
	defaultWindow time.Duration
	mu          sync.RWMutex
}

// NewEnhancedRateLimiter creates a new enhanced rate limiter
func NewEnhancedRateLimiter(redis *redis.Client) *EnhancedRateLimiter {
	return &EnhancedRateLimiter{
		redis:         redis,
		defaultRate:   100,
		defaultWindow: time.Minute,
	}
}

// IsAllowed checks if a request is allowed based on rate limiting
func (erl *EnhancedRateLimiter) IsAllowed(key string, rate int, window time.Duration) bool {
	ctx := context.Background()
	
	if rate <= 0 {
		rate = erl.defaultRate
	}
	if window <= 0 {
		window = erl.defaultWindow
	}
	
	now := time.Now()
	windowStart := now.Add(-window)
	
	pipe := erl.redis.Pipeline()
	
	// Use sliding window log approach
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	
	// Remove old entries
	pipe.ZRemRangeByScore(ctx, rateLimitKey, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	
	// Count current entries
	countCmd := pipe.ZCard(ctx, rateLimitKey)
	
	// Add current request
	pipe.ZAdd(ctx, rateLimitKey, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})
	
	// Set expiration
	pipe.Expire(ctx, rateLimitKey, window+time.Minute)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Rate limiter error: %v", err)
		return true // Allow on error
	}
	
	count, err := countCmd.Result()
	if err != nil {
		return true // Allow on error
	}
	
	return count < int64(rate)
}

// SetDefaultLimits sets the default rate limiting parameters
func (erl *EnhancedRateLimiter) SetDefaultLimits(rate int, window time.Duration) {
	erl.mu.Lock()
	defer erl.mu.Unlock()
	
	erl.defaultRate = rate
	erl.defaultWindow = window
}

// GetRemainingRequests gets the remaining requests for a key
func (erl *EnhancedRateLimiter) GetRemainingRequests(key string, rate int, window time.Duration) int {
	ctx := context.Background()
	
	if rate <= 0 {
		rate = erl.defaultRate
	}
	if window <= 0 {
		window = erl.defaultWindow
	}
	
	windowStart := time.Now().Add(-window)
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	
	// Count current entries
	count, err := erl.redis.ZCount(ctx, rateLimitKey, 
		fmt.Sprintf("%d", windowStart.UnixNano()), "+inf").Result()
	if err != nil {
		return rate // Return max on error
	}
	
	remaining := rate - int(count)
	if remaining < 0 {
		remaining = 0
	}
	
	return remaining
}

// Webhook Failover System
type FailoverManager struct {
	primaryEndpoints   []string
	secondaryEndpoints []string
	currentMode       string // "primary", "secondary", "both"
	healthChecker     *HealthChecker
	mu                sync.RWMutex
}

type HealthChecker struct {
	timeout     time.Duration
	checkInterval time.Duration
	healthStatus map[string]bool
	mu          sync.RWMutex
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager() *FailoverManager {
	return &FailoverManager{
		primaryEndpoints:   make([]string, 0),
		secondaryEndpoints: make([]string, 0),
		currentMode:       "primary",
		healthChecker: &HealthChecker{
			timeout:       10 * time.Second,
			checkInterval: 5 * time.Minute,
			healthStatus:  make(map[string]bool),
		},
	}
}

// AddPrimaryEndpoint adds a primary endpoint
func (fm *FailoverManager) AddPrimaryEndpoint(endpoint string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.primaryEndpoints = append(fm.primaryEndpoints, endpoint)
}

// AddSecondaryEndpoint adds a secondary endpoint
func (fm *FailoverManager) AddSecondaryEndpoint(endpoint string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.secondaryEndpoints = append(fm.secondaryEndpoints, endpoint)
}

// GetActiveEndpoints returns the currently active endpoints
func (fm *FailoverManager) GetActiveEndpoints() []string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	switch fm.currentMode {
	case "primary":
		return fm.primaryEndpoints
	case "secondary":
		return fm.secondaryEndpoints
	case "both":
		endpoints := make([]string, 0, len(fm.primaryEndpoints)+len(fm.secondaryEndpoints))
		endpoints = append(endpoints, fm.primaryEndpoints...)
		endpoints = append(endpoints, fm.secondaryEndpoints...)
		return endpoints
	default:
		return fm.primaryEndpoints
	}
}

// CheckFailover checks if failover is needed
func (fm *FailoverManager) CheckFailover() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	primaryHealthy := fm.areEndpointsHealthy(fm.primaryEndpoints)
	secondaryHealthy := fm.areEndpointsHealthy(fm.secondaryEndpoints)
	
	previousMode := fm.currentMode
	
	if primaryHealthy && fm.currentMode != "primary" {
		// Primary is healthy, switch back
		fm.currentMode = "primary"
	} else if !primaryHealthy && secondaryHealthy && fm.currentMode == "primary" {
		// Primary is unhealthy, failover to secondary
		fm.currentMode = "secondary"
	} else if !primaryHealthy && !secondaryHealthy {
		// Both unhealthy, try both
		fm.currentMode = "both"
	}
	
	if previousMode != fm.currentMode {
		log.Printf("Failover mode changed from %s to %s", previousMode, fm.currentMode)
	}
}

// areEndpointsHealthy checks if any endpoint in the list is healthy
func (fm *FailoverManager) areEndpointsHealthy(endpoints []string) bool {
	fm.healthChecker.mu.RLock()
	defer fm.healthChecker.mu.RUnlock()
	
	for _, endpoint := range endpoints {
		if healthy, exists := fm.healthChecker.healthStatus[endpoint]; exists && healthy {
			return true
		}
	}
	return false
}

// StartHealthChecks starts periodic health checks
func (fm *FailoverManager) StartHealthChecks() {
	go func() {
		ticker := time.NewTicker(fm.healthChecker.checkInterval)
		defer ticker.Stop()
		
		for range ticker.C {
			fm.performHealthChecks()
			fm.CheckFailover()
		}
	}()
}

// performHealthChecks performs health checks on all endpoints
func (fm *FailoverManager) performHealthChecks() {
	allEndpoints := make([]string, 0, len(fm.primaryEndpoints)+len(fm.secondaryEndpoints))
	allEndpoints = append(allEndpoints, fm.primaryEndpoints...)
	allEndpoints = append(allEndpoints, fm.secondaryEndpoints...)
	
	for _, endpoint := range allEndpoints {
		go func(ep string) {
			healthy := fm.checkEndpointHealth(ep)
			
			fm.healthChecker.mu.Lock()
			fm.healthChecker.healthStatus[ep] = healthy
			fm.healthChecker.mu.Unlock()
		}(endpoint)
	}
}

// checkEndpointHealth checks the health of a single endpoint
func (fm *FailoverManager) checkEndpointHealth(endpoint string) bool {
	client := &http.Client{Timeout: fm.healthChecker.timeout}
	
	resp, err := client.Head(endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode < 400
}

// GetHealthStatus returns the health status of all endpoints
func (fm *FailoverManager) GetHealthStatus() map[string]bool {
	fm.healthChecker.mu.RLock()
	defer fm.healthChecker.mu.RUnlock()
	
	status := make(map[string]bool)
	for endpoint, healthy := range fm.healthChecker.healthStatus {
		status[endpoint] = healthy
	}
	
	return status
}