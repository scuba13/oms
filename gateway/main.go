package main

import (
	"context"
	"net/http"

	//_ "github.com/joho/godotenv/autoload"
	common "github.com/scuba13/oms/common"
	"github.com/scuba13/oms/gateway/gateway"
	"log"
)

var (
	serviceName = "gateway"
	consulAddr  = common.EnvString("CONSUL_ADDR", "localhost:8500")
)

func main() {
	// Initialize logging
	logger, err := common.SetupLogging()
	if err != nil {
		log.Fatalf("Failed to set up logging: %v", err)
	}
	defer logger.Sync()

	logger.Sugar().Infof("Initializing service: %s", serviceName)

	// Create context with cancellation
	logger.Sugar().Info("Creating context with cancellation")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger.Sugar().Info("Context created")

	// Set up Consul registry
	registry, err := common.SetupConsul(consulAddr, serviceName)
	if err != nil {
		logger.Sugar().Fatalf("Failed to set up Consul registry: %v", err)
	}

	// Define configuration keys to fetch
	keys := []string{"gateway/HTTP_ADDR", "common/JAEGER_ADDR"}
	// Fetch configuration values from Consul
	config, err := common.FetchConfiguration(ctx, registry, keys)
	if err != nil {
		logger.Sugar().Fatalf("Failed to fetch configuration: %v", err)
	}
	httpAddr := config["gateway/HTTP_ADDR"]
	jaegerAddr := config["common/JAEGER_ADDR"]

	// Set up tracing
	if err := common.SetGlobalTracer(ctx, serviceName, jaegerAddr); err != nil {
		logger.Sugar().Fatalf("Failed to set global tracer: %v", err)
	}

	// Register service and start health check routine
	instanceID, cancelMonitor, err := common.RegisterAndMonitorService(ctx, registry, serviceName, httpAddr)
	if err != nil {
		logger.Sugar().Fatalf("Failed to register service with Consul: %v", err)
	}
	logger.Sugar().Infof("Service registered with Consul: %s", instanceID)
	defer cancelMonitor()

	// Set up HTTP server
	mux := http.NewServeMux()
	ordersGateway := gateway.NewGRPCGateway(registry)
	handler := NewHandler(ordersGateway)
	handler.registerRoutes(mux)

	server := common.SetupHTTPServer(httpAddr, mux)

	// Handle graceful shutdown
	common.HandleGracefulShutdown(ctx, cancel, server)
}
