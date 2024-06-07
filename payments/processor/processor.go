package processor

import pb "github.com/scuba13/oms/common/api"

type PaymentProcessor interface {
	CreatePaymentLink(*pb.Order) (string, error)
}
