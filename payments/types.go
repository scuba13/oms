package main

import (
	"context"

	pb "github.com/scuba13/oms/commons/api"
)

type PaymentsService interface {
	CreatePayment(context.Context, *pb.Order) (string, error)
}
