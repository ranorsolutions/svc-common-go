package service

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ranorsolutions/http-common-go/pkg/db/postgres"
	logs "github.com/ranorsolutions/http-common-go/pkg/log/logger"
	"github.com/ranorsolutions/svc-common-go/pkg/route"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	connectPostgres = postgres.Connect
	dialGRPC        = grpc.Dial
)

type Service struct {
	DB                 *sql.DB
	ServiceConnections map[string]*grpc.ClientConn
	Services           map[string]interface{}
	Logger             *logs.Logger
	Port               string
	HTTPHandlers       []*route.Handler
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
	db, err := connectPostgres(connString)
	if err != nil {
		// Avoid trying to log or access db if nil
		if logger != nil {
			logger.Error("failed to connect to database: %v", err)
		} else {
			log.Printf("failed to connect to database: %v", err)
		}
		return nil, fmt.Errorf("failed to create db connection: %v", err)
	}

	logger.Info("Connected to database %s", connString.HostString())

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
	raw := os.Getenv("SERVICE_DEPS")
	if raw != "" {
		for _, dep := range strings.Split(raw, ",") {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			parts := strings.SplitN(dep, "@", 2)
			if len(parts) != 2 {
				continue
			}
			name, addr := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if name == "" || addr == "" {
				continue
			}

			conn, err := dialGRPC(addr, grpcOptions...)
			if err != nil {
				return nil, fmt.Errorf("failed to dial %s: %w", name, err)
			}
			if conn == nil { // <- ensure non-nil
				return nil, fmt.Errorf("failed to dial %s: got nil connection", name)
			}

			services[name] = conn

			// Don't touch connectivity internals (mocks can panic).
			// If you still want a log, keep it best-effort and guarded.
			// defer func() {
			//     if r := recover(); r != nil {
			//         logger.Warn("skipped connectivity inspection for %s", name)
			//     }
			// }()
			// _ = conn.GetState()
			logger.Info("Connected to %s service", name)
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
