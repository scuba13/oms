package gateway

import (
	"context"

	pb "github.com/scuba13/oms/common/api"
)

type OrdersGateway interface {
	CreateOrder(context.Context, *pb.CreateOrderRequest) (*pb.Order, error)
	GetOrder(ctx context.Context, orderID, customerID string)(*pb.Order, error)
}
