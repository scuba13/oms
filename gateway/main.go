package main

import (
	"context"
	"log"
	"net/http"
	"time"

	_ "github.com/joho/godotenv/autoload"
	common "github.com/scuba13/oms/commons"
	"github.com/scuba13/oms/commons/discovery"
	"github.com/scuba13/oms/commons/discovery/consul"
	"github.com/scuba13/oms/gateway/gateway"
)

var (
	serviceName = "gateway"
	httpAddr    = common.EnvString("HTTP_ADDR", ":8080")
	consulAddr  = common.EnvString("CONSUL_ADDR", "localhost:8500")
	jaegerAddr  = common.EnvString("JAEGER_ADDR", "localhost:4318")
)

func main() {
	log.Printf("Initializing service: %s", serviceName)

	err := common.SetGlobalTracer(context.TODO(), serviceName, jaegerAddr)
	if err != nil {
		log.Fatalf("Failed to set global tracer: %v", err)
	}

	registry, err := consul.NewRegistry(consulAddr, serviceName)
	if err != nil {
		log.Fatalf("Failed to create consul registry: %v", err)
	}

	ctx := context.Background()
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, httpAddr); err != nil {
		log.Fatalf("Failed to register service with consul: %v", err)
	}

	go func() {
		for {
			if err := registry.HealthCheck(instanceID, serviceName); err != nil {
				log.Printf("Failed health check: %v", err)
			}
			time.Sleep(time.Second * 1)
		}
	}()

	defer func() {
		if err := registry.Deregister(ctx, instanceID, serviceName); err != nil {
			log.Printf("Failed to deregister service: %v", err)
		}
	}()

	mux := http.NewServeMux()

	ordersGateway := gateway.NewGRPCGateway(registry)
	if err != nil {
		log.Fatalf("Failed to create orders gateway: %v", err)
	}

	handler := NewHandler(ordersGateway)
	handler.registerRoutes(mux)

	log.Printf("Starting HTTP server at %s", httpAddr)

	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
