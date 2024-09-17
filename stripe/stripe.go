package paypal

import (
	"context"
	"log"
	"net/http"
	"os"

	"achan.moe/auth"
	"github.com/labstack/echo/v4"
	"github.com/plutov/paypal/v4"
)

var client *paypal.Client

func init() {
	var err error
	client, err = paypal.NewClient(os.Getenv("PAYPAL_CLIENT_ID"), os.Getenv("PAYPAL_SECRET"), paypal.APIBaseLive)
	if err != nil {
		log.Fatalf("Failed to create PayPal client: %v", err)
	}
	client.SetLog(os.Stdout) // Set log to terminal stdout
}

func SuccessHandler(c echo.Context) error {
	orderID := c.QueryParam("orderID")
	if orderID == "" {
		return c.String(http.StatusBadRequest, "Missing order ID")
	}

	// Retrieve the order from PayPal
	order, err := client.GetOrder(context.Background(), orderID)
	if err != nil {
		log.Printf("Failed to retrieve order: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve order")
	}

	transaction := order.PurchaseUnits[0].Payments.Captures[0]
	if transaction.ID == "" {
		return c.String(http.StatusBadRequest, "Missing transaction ID")
	}
	payer := order.Payer
	if payer == nil || payer.EmailAddress == "" {
		return c.String(http.StatusBadRequest, "Missing payer details")
	}
	auth.NewPremiumUser(c, payer.EmailAddress, transaction.ID)

	log.Printf("Payment successful for customer: %s, transaction ID: %s", payer.EmailAddress, transaction.ID)

	return c.String(http.StatusOK, "Payment successful! You should be redirected shortly.")
}
