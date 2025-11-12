package firebase

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"github.com/stretchr/testify/assert"
)

// --- Mock Firebase Auth client ---

type mockAuthClient struct {
	verifyFunc func(context.Context, string) (*auth.Token, error)
}

func (m *mockAuthClient) VerifyIDToken(ctx context.Context, token string) (*auth.Token, error) {
	return m.verifyFunc(ctx, token)
}

// --- Helpers ---

func newBaseService(t *testing.T) *service.Service {
	if os.Getenv("CI") == "true" {
		return service.NewMock()
	}

	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "test")
	os.Setenv("DB_PASSWORD", "test")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_SSL_MODE", "disable")
	os.Setenv("SERVICE", "test-service")
	os.Setenv("VERSION", "1.0.0")

	svc, err := service.New()
	if err != nil {
		t.Fatalf("failed to create base service: %v", err)
	}
	return svc
}

// --- Tests ---

func TestGetConfigFromEnv(t *testing.T) {
	t.Setenv("FIREBASE_CREDENTIALS", "/tmp/creds.json")
	t.Setenv("FIREBASE_PROJECT_ID", "test-proj")

	cfg := getConfigFromEnv()
	assert.Equal(t, "/tmp/creds.json", cfg.CredentialsPath)
	assert.Equal(t, "test-proj", cfg.ProjectID)
}

func TestVerifyToken_Success(t *testing.T) {
	svc := newBaseService(t)
	fs := &FirebaseService{
		Base: svc,
		Auth: &mockAuthClient{
			verifyFunc: func(ctx context.Context, token string) (*auth.Token, error) {
				return &auth.Token{UID: "user123", Claims: map[string]interface{}{"role": "admin"}}, nil
			},
		},
	}

	tok, err := fs.VerifyToken(context.Background(), "valid-token")
	assert.NoError(t, err)
	assert.Equal(t, "user123", tok.UID)
	assert.Equal(t, "admin", tok.Claims["role"])
}

func TestVerifyToken_Invalid(t *testing.T) {
	svc := newBaseService(t)
	fs := &FirebaseService{
		Base: svc,
		Auth: &mockAuthClient{
			verifyFunc: func(ctx context.Context, token string) (*auth.Token, error) {
				return nil, errors.New("invalid token")
			},
		},
	}

	tok, err := fs.VerifyToken(context.Background(), "bad-token")
	assert.Error(t, err)
	assert.Nil(t, tok)
}

func TestFirebaseAuthMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newBaseService(t)
	fs := &FirebaseService{
		Base: svc,
		Auth: &mockAuthClient{
			verifyFunc: func(ctx context.Context, token string) (*auth.Token, error) {
				return &auth.Token{UID: "abc123"}, nil
			},
		},
	}

	r := gin.New()
	r.Use(fs.FirebaseAuthMiddleware())
	r.GET("/secure", func(c *gin.Context) {
		user := GetFirebaseUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.JSON(200, gin.H{"uid": user.UID})
	})

	req, _ := http.NewRequest("GET", "/secure", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Body.String(), "abc123")
}

func TestFirebaseAuthMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := &FirebaseService{
		Base: newBaseService(t),
	}

	r := gin.New()
	r.Use(fs.FirebaseAuthMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	req, _ := http.NewRequest("GET", "/secure", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)
	assert.Equal(t, 401, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing Authorization")
}

func TestFirebaseAuthMiddleware_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := &FirebaseService{
		Base: newBaseService(t),
	}

	r := gin.New()
	r.Use(fs.FirebaseAuthMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	req, _ := http.NewRequest("GET", "/secure", nil)
	req.Header.Set("Authorization", "Token invalid-format")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, 401, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid Authorization format")
}

func TestFirebaseAuthMiddleware_VerifyFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newBaseService(t)
	fs := &FirebaseService{
		Base: svc,
		Auth: &mockAuthClient{
			verifyFunc: func(ctx context.Context, token string) (*auth.Token, error) {
				return nil, errors.New("bad token")
			},
		},
	}

	r := gin.New()
	r.Use(fs.FirebaseAuthMiddleware())
	r.GET("/secure", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/secure", nil)
	req.Header.Set("Authorization", "Bearer bad-token")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, 401, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestGetFirebaseUser_Helper(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	tok := &auth.Token{UID: "user-xyz"}
	c.Set("firebaseUser", tok)

	res := GetFirebaseUser(c)
	assert.NotNil(t, res)
	assert.Equal(t, "user-xyz", res.UID)
}

func TestGetFirebaseUser_NoUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	res := GetFirebaseUser(c)
	assert.Nil(t, res)
}
