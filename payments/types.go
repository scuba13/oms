package main

import (
	"context"

	pb "github.com/scuba13/oms/common/api"
)

type PaymentsService interface {
	CreatePayment(context.Context, *pb.Order) (string, error)
}
