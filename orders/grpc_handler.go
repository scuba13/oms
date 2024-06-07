package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	pb "github.com/scuba13/oms/common/api"
	"github.com/scuba13/oms/common/broker"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
)

type grpcHandler struct {
	pb.UnimplementedOrderServiceServer

	service OrdersService
	channel *amqp.Channel
}

func NewGRPCHandler(grpcServer *grpc.Server, service OrdersService, channel *amqp.Channel) {
	handler := &grpcHandler{
		service: service,
		channel: channel,
	}
	pb.RegisterOrderServiceServer(grpcServer, handler)
}

func (h *grpcHandler) UpdateOrder(ctx context.Context, p *pb.Order) (*pb.Order, error) {
	return h.service.UpdateOrder(ctx, p)
}

func (h *grpcHandler) GetOrder(ctx context.Context, p *pb.GetOrderRequest) (*pb.Order, error) {
	return h.service.GetOrder(ctx, p)
}

func (h *grpcHandler) CreateOrder(ctx context.Context, p *pb.CreateOrderRequest) (*pb.Order, error) {
	log.Println("Starting CreateOrder process")

	// Step 1: Declare the queue
	log.Println("Declaring the queue")
	q, err := h.channel.QueueDeclare(broker.OrderCreatedEvent, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Step 2: Start tracing span
	log.Println("Starting tracing span for AMQP")
	tr := otel.Tracer("amqp")
	amqpContext, messageSpan := tr.Start(ctx, fmt.Sprintf("AMQP - publish - %s", q.Name))
	defer messageSpan.End()

	// Step 3: Validate order
	log.Println("Validating order")
	items, err := h.service.ValidateOrder(amqpContext, p)
	if err != nil {
		log.Printf("Order validation failed: %v", err)
		return nil, err
	}

	// Step 4: Create order
	log.Println("Creating order")
	o, err := h.service.CreateOrder(amqpContext, p, items)
	if err != nil {
		log.Printf("Order creation failed: %v", err)
		return nil, err
	}

	// Step 5: Marshal order to JSON
	log.Println("Marshalling order to JSON")
	marshalledOrder, err := json.Marshal(o)
	if err != nil {
		log.Printf("Failed to marshal order: %v", err)
		return nil, err
	}

	// Step 6: Inject headers
	log.Println("Injecting AMQP headers")
	headers := broker.InjectAMQPHeaders(amqpContext)

	// Step 7: Publish message
	log.Println("Publishing message to queue")
	err = h.channel.PublishWithContext(amqpContext, "", q.Name, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         marshalledOrder,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
	})
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return nil, err
	}

	log.Println("Order created successfully")
	return o, nil
}
