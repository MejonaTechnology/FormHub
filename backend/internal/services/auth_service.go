package services

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"formhub/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db        *sql.DB
	redis     *redis.Client
	jwtSecret []byte
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

func NewAuthService(db *sql.DB, redis *redis.Client, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		redis:     redis,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthService) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	
	if exists {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Company:   req.Company,
		PlanType:  "free",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, company, 
			plan_type, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = s.db.Exec(query,
		user.ID, user.Email, user.Password, user.FirstName, user.LastName,
		user.Company, user.PlanType, user.IsActive, user.CreatedAt, user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create default API key
	_, err = s.CreateAPIKey(user.ID, "Default API Key")
	if err != nil {
		// Log error but don't fail registration
		fmt.Printf("Failed to create default API key: %v\n", err)
	}

	// Clear password before returning
	user.Password = ""

	return &models.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, company, 
			plan_type, is_active, created_at, updated_at
		FROM users WHERE email = $1 AND is_active = true
	`

	err := s.db.QueryRow(query, req.Email).Scan(
		&user.ID, &user.Email, &user.Password, &user.FirstName, &user.LastName,
		&user.Company, &user.PlanType, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Clear password before returning
	user.Password = ""

	return &models.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, first_name, last_name, company, plan_type, is_active,
			created_at, updated_at
		FROM users WHERE id = $1 AND is_active = true
	`

	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName,
		&user.Company, &user.PlanType, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *AuthService) CreateAPIKey(userID uuid.UUID, name string) (*models.APIKey, error) {
	// Generate a random API key
	key := uuid.New().String() + "-" + uuid.New().String()
	keyHash := fmt.Sprintf("%x", md5.Sum([]byte(key)))

	apiKey := &models.APIKey{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		KeyHash:     keyHash,
		Key:         key, // Only shown when created
		Permissions: "form_submit",
		RateLimit:   1000, // requests per minute
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO api_keys (id, user_id, name, key_hash, permissions, rate_limit,
			is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.db.Exec(query,
		apiKey.ID, apiKey.UserID, apiKey.Name, apiKey.KeyHash,
		apiKey.Permissions, apiKey.RateLimit, apiKey.IsActive,
		apiKey.CreatedAt, apiKey.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return apiKey, nil
}

func (s *AuthService) GetUserAPIKeys(userID uuid.UUID) ([]models.APIKey, error) {
	query := `
		SELECT id, user_id, name, permissions, rate_limit, is_active,
			last_used_at, created_at, updated_at
		FROM api_keys WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []models.APIKey
	for rows.Next() {
		var apiKey models.APIKey
		err := rows.Scan(
			&apiKey.ID, &apiKey.UserID, &apiKey.Name, &apiKey.Permissions,
			&apiKey.RateLimit, &apiKey.IsActive, &apiKey.LastUsedAt,
			&apiKey.CreatedAt, &apiKey.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

func (s *AuthService) DeleteAPIKey(keyID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE api_keys SET is_active = false, updated_at = $1
		WHERE id = $2 AND user_id = $3
	`

	result, err := s.db.Exec(query, time.Now(), keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("API key not found or unauthorized")
	}

	return nil
}

func (s *AuthService) generateTokens(user *models.User) (string, string, error) {
	// Access token (15 minutes)
	accessClaims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token (7 days)
	refreshClaims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}