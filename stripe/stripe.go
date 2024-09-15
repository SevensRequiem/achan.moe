package stripe

import (
	"log"
	"net/http"
	"os"

	"achan.moe/auth"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
)

func init() {
	// Set your Stripe secret API key.
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}

func SuccessHandler(c echo.Context) error {
	id := c.QueryParam("id")
	if id == "" {
		return c.String(http.StatusBadRequest, "Missing checkout ID")
	}

	// Retrieve the session from Stripe
	s, err := session.Get(id, nil)
	if err != nil {
		log.Printf("Failed to retrieve session: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	auth.NewPremiumUser(c)
	log.Printf("Payment successful: %+v", s)

	return c.String(http.StatusOK, "Payment successful! You should be redirected shortly.")
}
