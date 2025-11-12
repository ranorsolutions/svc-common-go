package firebase

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	fb "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"google.golang.org/api/option"
)

// AuthAPI defines the subset of Firebase Auth methods we use.
// This makes it mockable in tests.
type AuthAPI interface {
	VerifyIDToken(ctx context.Context, token string) (*auth.Token, error)
}

// FirebaseService wraps a base service with Firebase integration.
type FirebaseService struct {
	Base   *service.Service
	App    *fb.App
	Auth   AuthAPI
	Config *FirebaseConfig
}

// FirebaseConfig defines the Firebase configuration.
type FirebaseConfig struct {
	CredentialsPath string
	ProjectID       string
}

// NewFirebaseService creates a new Firebase-integrated service using the base service.
func NewFirebaseService(base *service.Service, cfg *FirebaseConfig) (*FirebaseService, error) {
	if base == nil {
		return nil, fmt.Errorf("base service is required")
	}

	if cfg == nil {
		cfg = getConfigFromEnv()
	}

	opts := []option.ClientOption{}
	if cfg.CredentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsPath))
	}

	if cfg.ProjectID != "" {
		os.Setenv("GOOGLE_CLOUD_PROJECT", cfg.ProjectID)
	}

	app, err := fb.NewApp(context.Background(), nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	authClient, err := app.Auth(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase Auth: %w", err)
	}

	base.Logger.Info("Firebase initialized for project %s", cfg.ProjectID)

	return &FirebaseService{
		Base:   base,
		App:    app,
		Auth:   authClient,
		Config: cfg,
	}, nil
}

func getConfigFromEnv() *FirebaseConfig {
	return &FirebaseConfig{
		CredentialsPath: os.Getenv("FIREBASE_CREDENTIALS"),
		ProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
	}
}

// VerifyToken verifies and decodes a Firebase ID token.
func (fs *FirebaseService) VerifyToken(ctx context.Context, token string) (*auth.Token, error) {
	tok, err := fs.Auth.VerifyIDToken(ctx, token)
	if err != nil {
		fs.Base.Logger.Error("failed to verify Firebase token: %v", err)
		return nil, fmt.Errorf("invalid Firebase token: %w", err)
	}
	return tok, nil
}

// FirebaseAuthMiddleware returns a Gin middleware that validates Firebase tokens.
func (fs *FirebaseService) FirebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization format"})
			return
		}

		tokenStr := parts[1]
		tok, err := fs.VerifyToken(c.Request.Context(), tokenStr)
		if err != nil {
			fs.Base.Logger.Warn("unauthorized request: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set("firebaseUser", tok)
		c.Next()
	}
}

// GetFirebaseUser retrieves the authenticated Firebase user from the Gin context.
func GetFirebaseUser(c *gin.Context) *auth.Token {
	if v, ok := c.Get("firebaseUser"); ok {
		if tok, ok := v.(*auth.Token); ok {
			return tok
		}
	}
	return nil
}
