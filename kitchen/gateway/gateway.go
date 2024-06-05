package gateway

import (
	"context"

	pb "github.com/scuba13/oms/commons/api"
)

type KitchenGateway interface {
	UpdateOrder(context.Context, *pb.Order) error
}
