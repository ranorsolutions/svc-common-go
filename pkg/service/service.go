package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ranorsolutions/http-common-go/pkg/db/postgres"
	logs "github.com/ranorsolutions/http-common-go/pkg/log/logger"
	"github.com/ranorsolutions/svc-common-go/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Service struct {
	DB                 *sql.DB
	ServiceConnections map[string]*grpc.ClientConn
	Services           map[string]interface{}
	Logger             *logs.Logger
	Port               string
	HTTPHandlers       []*types.HTTPHandler
}

type ServiceOption struct {
	GRPCCredential credentials.TransportCredentials
}

// New -- Create a firebase app
func New(serviceOpts ...ServiceOption) (*Service, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	// Create the service logger
	logger, err := logs.New(os.Getenv("SERVICE"), os.Getenv("VERSION"), os.Getenv("IS_TERMINAL") != "true")
	if err != nil {
		log.Fatalf("unable to create service logger")
	}

	// Connect to the Database
	connString := postgres.GetURIFromEnv()
	db, err := postgres.Connect(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create db connection: %v", err)
	}
	logger.Info(fmt.Sprintf("Connected to database %s", connString.HostString()))

	grpcOptions := []grpc.DialOption{}
	if len(serviceOpts) > 0 {
		for _, option := range serviceOpts {
			if option.GRPCCredential != nil {
				grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(option.GRPCCredential))
			}
		}
	} else {
		grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Parse the service dependencies
	services := map[string]*grpc.ClientConn{}
	parsed := strings.Split(os.Getenv("SERVICE_DEPS"), ",")
	for _, service := range parsed {
		parsedService := strings.Split(service, "@")
		if len(parsedService) == 2 {
			conn, err := grpc.Dial(parsedService[1], grpcOptions...)
			if err != nil {
				return nil, err
			}
			services[parsedService[0]] = conn
			conn.WaitForStateChange(context.Background(), connectivity.Connecting)
			logger.Info("Connected to %s service: %s", parsedService[0], conn.GetState())
		}
	}

	// Create a new FirebaseApp instance
	service := &Service{
		DB:                 db,
		ServiceConnections: services,
		Logger:             logger,
		Port:               port,
	}

	return service, nil
}

func (s *Service) HandleErr(c *gin.Context, err error, message string, code int) {
	s.Logger.Error(message)
	if message == "" {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {

		c.JSON(code, gin.H{"error": err.Error(), "details": message})
	}
}
