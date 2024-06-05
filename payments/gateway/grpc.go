package gateway

import (
	"context"
	"log"

	pb "github.com/scuba13/oms/commons/api"
	"github.com/scuba13/oms/commons/discovery"
)

type gateway struct {
	registry discovery.Registry
}

func NewGateway(registry discovery.Registry) *gateway {
	return &gateway{registry}
}

func (g *gateway) UpdateOrderAfterPaymentLink(ctx context.Context, orderID, paymentLink string) error {
	conn, err := discovery.ServiceConnection(context.Background(), "orders", g.registry)
	if err != nil {
		log.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()

	ordersClient := pb.NewOrderServiceClient(conn)

	_, err = ordersClient.UpdateOrder(ctx, &pb.Order{
		ID:          orderID,
		Status:      "waiting_payment",
		PaymentLink: paymentLink,
	})
	return err
}
