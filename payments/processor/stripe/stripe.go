package stripe

import (
	"fmt"
	"log"

	common "github.com/scuba13/oms/common"
	pb "github.com/scuba13/oms/common/api"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
)

var gatewayHTTPAddr = common.EnvString("GATEWAY_HTTP_ADDRESS", "http://localhost:8080")

type Stripe struct{}

func NewProcessor() *Stripe {
	// Configure a chave de API do Stripe diretamente no código
	stripe.Key = "sk_test_51POf3ARwn3euj82DTKV1gVEqsvjI1p51zQVN5MwmLhcdRh7pfWhU1G0ADka6aK1x5S8DmYaeREEdaGAcESyn7Q9o00d7jecplw"
	return &Stripe{}
}


func (s *Stripe) CreatePaymentLink(o *pb.Order) (string, error) {
	log.Printf("Creating payment link for order %v", o)

	gatewaySuccessURL := fmt.Sprintf("%s/success.html?customerID=%s&orderID=%s", gatewayHTTPAddr, o.CustomerID, o.ID)
	gatewayCancelURL := fmt.Sprintf("%s/cancel.html", gatewayHTTPAddr)

	items := []*stripe.CheckoutSessionLineItemParams{}
	for _, item := range o.Items {
		items = append(items, &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(item.PriceID),
			Quantity: stripe.Int64(int64(item.Quantity)),
		})
	}

	params := &stripe.CheckoutSessionParams{
		Metadata: map[string]string{
			"orderID":    o.ID,
			"customerID": o.CustomerID,
		},
		LineItems:  items,
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(gatewaySuccessURL),
		CancelURL:  stripe.String(gatewayCancelURL),
	}

	result, err := session.New(params)
	if err != nil {
		return "", nil
	}

	return result.URL, nil
}
