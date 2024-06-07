package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "github.com/joho/godotenv/autoload"
	common "github.com/scuba13/oms/commons"
	"github.com/scuba13/oms/commons/discovery"
	"github.com/scuba13/oms/commons/discovery/consul"
	"github.com/scuba13/oms/gateway/gateway"
)

var (
	serviceName = "gateway"
	consulAddr  = common.EnvString("CONSUL_ADDR", "localhost:8500")
)

func main() {
	log.Printf("Initializing service: %s", serviceName)

	// Set up Consul registry
	registry, err := consul.NewRegistry(consulAddr, serviceName)
	if err != nil {
		log.Fatalf("Failed to create Consul registry: %v", err)
	}

	// Fetch HTTP_ADDR and JAEGER_ADDR from Consul
	ctx := context.Background()
	httpAddr, err := registry.GetValue(ctx, "HTTP_ADDR")
	if err != nil {
		log.Fatalf("Failed to get HTTP_ADDR from Consul: %v", err)
	}

	jaegerAddr, err := registry.GetValue(ctx, "JAEGER_ADDR")
	if err != nil {
		log.Fatalf("Failed to get JAEGER_ADDR from Consul: %v", err)
	}

	log.Printf("Using HTTP_ADDR: %s", httpAddr)
	log.Printf("Using JAEGER_ADDR: %s", jaegerAddr)

	// Set up tracing
	if err := common.SetGlobalTracer(ctx, serviceName, jaegerAddr); err != nil {
		log.Fatalf("Failed to set global tracer: %v", err)
	}

	// Register service with Consul
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, httpAddr); err != nil {
		log.Fatalf("Failed to register service with Consul: %v", err)
	}

	// Start health check routine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := registry.HealthCheck(instanceID, serviceName); err != nil {
					log.Printf("Failed health check: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	defer func() {
		if err := registry.Deregister(ctx, instanceID, serviceName); err != nil {
			log.Printf("Failed to deregister service: %v", err)
		}
	}()

	// Set up HTTP server
	mux := http.NewServeMux()

	ordersGateway := gateway.NewGRPCGateway(registry)
	if err != nil {
		log.Fatalf("Failed to create orders gateway: %v", err)
	}

	handler := NewHandler(ordersGateway)
	handler.registerRoutes(mux)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Starting HTTP server at %s", httpAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
