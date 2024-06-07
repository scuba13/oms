package common

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/scuba13/oms/commons/broker"
	"github.com/scuba13/oms/commons/discovery"
	"github.com/scuba13/oms/commons/discovery/consul"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	amqp "github.com/rabbitmq/amqp091-go"
)

// SetupLogging initializes the global logger using a custom Zap configuration.
func SetupLogging() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "msg"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	zap.ReplaceGlobals(logger)
	return logger, nil
}

func SetupConsul(consulAddr, serviceName string) (*consul.Registry, error) {
	registry, err := consul.NewRegistry(consulAddr, serviceName)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func FetchConfiguration(ctx context.Context, registry *consul.Registry, keys []string) (map[string]string, error) {
	config := make(map[string]string)
	for _, key := range keys {
		value, err := registry.GetValue(ctx, key)
		if err != nil {
			return nil, err
		}
		config[key] = value
		zap.S().Infof("Fetched configuration for %s: %s", key, value)
	}
	return config, nil
}

func RegisterService(ctx context.Context, registry *consul.Registry, serviceName, addr string) (string, error) {
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, addr); err != nil {
		return "", err
	}
	return instanceID, nil
}

func StartHealthCheckRoutine(ctx context.Context, registry *consul.Registry, instanceID, serviceName string) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := registry.HealthCheck(instanceID, serviceName); err != nil {
					zap.S().Errorf("Failed health check: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func HandleGracefulShutdown(ctx context.Context, cancel context.CancelFunc, registry *consul.Registry, instanceID, serviceName string, servers ...*http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	zap.S().Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	for _, server := range servers {
		if server != nil {
			if err := server.Shutdown(shutdownCtx); err != nil {
				zap.S().Fatalf("Server forced to shutdown: %v", err)
			}
		}
	}

	if err := registry.Deregister(ctx, instanceID, serviceName); err != nil {
		zap.S().Errorf("Failed to deregister service: %v", err)
	}

	cancel()
	zap.S().Info("Server exiting")
}

func SetupGRPCServer(grpcAddr string, registry *consul.Registry, handler func(*grpc.Server)) (*grpc.Server, net.Listener) {
	grpcServer := grpc.NewServer()

	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		zap.S().Fatalf("Failed to listen: %v", err)
	}

	handler(grpcServer)

	go func() {
		zap.S().Infof("Starting gRPC server at %s", grpcAddr)
		if err := grpcServer.Serve(l); err != nil {
			zap.S().Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	return grpcServer, l
}

func SetupHTTPServer(httpAddr string, mux *http.ServeMux) *http.Server {
	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	go func() {
		zap.S().Infof("Starting HTTP server at %s", httpAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	return server
}

// SetupBroker initializes the broker connection.
func SetupBroker(amqpUser, amqpPass, amqpHost, amqpPort string) (*amqp.Channel, func() error) {
	ch, closeFunc := broker.Connect(amqpUser, amqpPass, amqpHost, amqpPort)
	return ch, closeFunc
}
