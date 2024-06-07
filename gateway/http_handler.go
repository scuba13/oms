package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	common "github.com/scuba13/oms/commons"
	pb "github.com/scuba13/oms/commons/api"
	"github.com/scuba13/oms/gateway/gateway"
	"go.opentelemetry.io/otel"
	otelCodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type handler struct {
	gateway gateway.OrdersGateway
}

func NewHandler(gateway gateway.OrdersGateway) *handler {
	return &handler{gateway}
}

func (h *handler) registerRoutes(mux *http.ServeMux) {
	// static folder serving
	mux.Handle("/", http.FileServer(http.Dir("public")))

	mux.HandleFunc("POST /api/customers/{customerID}/orders", h.handleCreateOrder)
	mux.HandleFunc("GET /api/customers/{customerID}/orders/{orderID}", h.handleGetOrder)
}

func (h *handler) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customerID")
	orderID := r.PathValue("orderID")

	// Create a tracer span
	tr := otel.Tracer("http")
	ctx, span := tr.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.RequestURI))
	defer span.End()

	o, err := h.gateway.GetOrder(ctx, orderID, customerID)
	rStatus := status.Convert(err)
	if rStatus != nil {
		span.SetStatus(otelCodes.Error, err.Error())

		if rStatus.Code() != codes.InvalidArgument {
			common.WriteError(w, http.StatusBadRequest, rStatus.Message())
			return
		}

		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, o)
}

func (h *handler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customerID")

	var req struct {
		Items []*pb.ItemsWithQuantity `json:"Items"`
	}

	if err := common.ReadJSON(r, &req); err != nil {
		common.WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	log.Printf("Request: %+v", req)

	if err := validateItems(req.Items); err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("Items: %+v", req.Items)

	// Create a tracer span
	tr := otel.Tracer("http")
	ctx, span := tr.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.RequestURI))
	defer span.End()

	o, err := h.gateway.CreateOrder(ctx, &pb.CreateOrderRequest{
		CustomerID: customerID,
		Items:      req.Items,
	})
	rStatus := status.Convert(err)
	if rStatus != nil {
		span.SetStatus(otelCodes.Error, err.Error())

		if rStatus.Code() != codes.InvalidArgument {
			common.WriteError(w, http.StatusBadRequest, rStatus.Message())
			return
		}

		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	res := &CreateOrderRequest{
		Order:         o,
		RedirectToURL: fmt.Sprintf("http://localhost:8080/success.html?customerID=%s&orderID=%s", o.CustomerID, o.ID),
	}

	common.WriteJSON(w, http.StatusOK, res)
}

func validateItems(items []*pb.ItemsWithQuantity) error {
	if len(items) == 0 {
		return common.ErrNoItems
	}

	for _, i := range items {
		if i.ID == "" {
			return errors.New("item ID is required")
		}

		if i.Quantity <= 0 {
			return errors.New("items must have a valid quantity")
		}
	}

	return nil
}
