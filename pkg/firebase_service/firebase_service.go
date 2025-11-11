package firebase_service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/ranorsolutions/svc-common-go/pkg/server"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"github.com/ranorsolutions/svc-common-go/pkg/types"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/api/option"
)

type FirebaseService struct {
	App        *firebase.App
	Auth       *auth.Client
	DB         *sql.DB
	Common     *service.Service
	Definition any
	Server     *server.Server
}

type AuthResponse struct {
	RefreshToken string `json:"refreshToken"`
	IDToken      string `json:"idToken"`
	SessionID    string `json:"sessionId"`
	AccessToken  string `json:"accessToken"`
}

// credentialsFile is the unmarshalled representation of a credentials file.
type credentialsFile struct {
	Type string `json:"type"` // serviceAccountKey or userCredentialsKey

	// Service Account fields
	ClientEmail  string `json:"client_email"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	TokenURL     string `json:"token_uri"`
	ProjectID    string `json:"project_id"`

	// User Credential fields
	// (These typically come from gcloud auth.)
	ClientSecret string `json:"client_secret"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
}

type Service[T any] interface {
	New(*FirebaseService) *T
}

func New[T any](s Service[T], isAuth bool) *FirebaseService {
	// Create the service from the common service module
	svc, err := service.New()
	if err != nil {
		log.Fatal("fatal error creating service", err)
	}

	// Append the service fields to the fire base service
	fbSvc := &FirebaseService{
		DB:     svc.DB,
		Common: svc,
	}

	if isAuth {
		app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsFile(os.Getenv("SERVICE_ACCOUNT_FILE")))
		if err != nil {
			log.Fatal("fatal error creating service", fmt.Errorf("error initializing app: %v", err))
		}

		// Create the firebase auth client
		client, err := app.Auth(context.Background())
		if err != nil {
			log.Fatal("fatal error creating service", fmt.Errorf("failed to create auth client: %v", err))
		}

		fbSvc.App = app
		fbSvc.Auth = client
	}

	// Create a new server
	srv := server.New(fbSvc.Common, os.Getenv("API_VERSION"))
	fbSvc.Server = srv

	// Register the gRPC Clients
	fbSvc.Common.Services = make(map[string]interface{})

	if s != nil {
		// Create the service def
		def := s.New(fbSvc)
		fbSvc.Definition = def
	}

	// Return the newly created firebase service
	return fbSvc
}

func (s *FirebaseService) SetRoutes(protected map[string]gin.HandlerFunc) {
	// Create the initial routes with any unprotected routes
	routes := []*types.HTTPHandler{
		{
			Type:    "GET",
			Path:    "/swagger/*any",
			Handler: []gin.HandlerFunc{ginSwagger.WrapHandler(swaggerFiles.Handler)},
		},
	}

	// Range over the protected routes to add the appropriate permission handler
	for route, handler := range protected {
		routeType, path := func() (string, string) {
			parts := strings.Split(route, "|")
			return parts[0], parts[1]
		}()

		routes = append(routes, &types.HTTPHandler{
			Type:    routeType,
			Path:    path,
			Handler: []gin.HandlerFunc{Auth([]string{}), handler},
		})
	}

	// Apply all the route handlers
	s.Common.HTTPHandlers = routes
}

func Auth(requiredPermissions []string) gin.HandlerFunc {
	client, err := kms.NewKeyManagementClient(context.Background())
	if err != nil {
		return nil
	}

	return func(c *gin.Context) {
		tokenString := ""
		authToken := c.GetHeader("Authorization")
		if authToken == "" {
			cookieResults, err := DecodeSession(c, client)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
			tokenString = cookieResults.AccessToken
		} else {
			parts := strings.Split(authToken, "Bearer ")
			tokenString = parts[1]
		}

		accessToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("AUTH_SECRET")), nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("error parsing token: %s", err.Error())})
			return
		}

		claims := accessToken.Claims.(jwt.MapClaims)
		if claims["disabled"] == true {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("user account %s is disabled", claims["uid"])})
			return
		}

		if claims["emailVerified"] == false {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("email for account %s is not verified", claims["uid"])})
			return
		}

		license, decodeErr := claims["license"].(map[string]interface{})
		if !decodeErr {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("user %s does not have a valid license", claims["uid"])})
			return
		}

		today := time.Now()
		startDate, err := time.Parse(time.RFC3339, license["startDate"].(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("user %s does not have a valid license: %s", claims["uid"], err.Error())})
			return
		}

		endDate, err := time.Parse(time.RFC3339, license["endDate"].(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("user %s does not have a valid license %s", claims["uid"], err.Error())})
			return
		}

		// Check for license that has not started
		if today.Before(startDate) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("license has not started for user %s", claims["uid"])})
			return
		}

		// Check for expired license
		if today.After(endDate) && !endDate.IsZero() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("license has expired for user %s", claims["uid"])})
			return
		}

		permissions, decodeErr := claims["permissions"].([]interface{})
		if !decodeErr {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("user %s does not have any permissions", claims["uid"])})
			return
		}

		hasAllRequired := true
		for _, required := range requiredPermissions {
			hasRequired := false
			for _, permission := range permissions {
				localPermission, ok := permission.(string)
				if !ok {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("error getting permissions for account %s", claims["uid"])})
					return
				}

				if localPermission == required {
					hasRequired = true
				}
			}

			hasAllRequired = hasRequired
		}

		if !hasAllRequired {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing required permissions to access this endpoint"})
			return
		}

		c.Next()
	}
}

func DecodeSession(c *gin.Context, client *kms.KeyManagementClient) (*AuthResponse, error) {
	cookie, err := c.Request.Cookie("auth-cookie")
	if err != nil {
		return nil, fmt.Errorf("missing cookie")
	}

	decodedCookie, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil, err
	}

	// Build the request.
	req := &kmspb.DecryptRequest{
		Name:       os.Getenv("AUTH_KEY"),
		Ciphertext: []byte(decodedCookie),
	}

	// Call the API.
	result, err := client.Decrypt(c, req)
	if err != nil {
		return nil, err
	}

	cookieResults := &AuthResponse{}
	json.Unmarshal(result.Plaintext, &cookieResults)

	return cookieResults, nil
}

func (s *FirebaseService) TODO() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"error": "not implemented"})
	}
}

func (s *FirebaseService) HandleErr(c *gin.Context, err error, code int, message ...string) {
	s.Common.Logger.Error(fmt.Sprintf("%s %s", err.Error(), strings.Join(message, " ")))
	if len(message) == 0 {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.JSON(code, gin.H{"error": err.Error(), "details": strings.Join(message, " ")})
	}
}
