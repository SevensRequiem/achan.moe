package store

import (
	"fmt"

	"achan.moe/stripe"
)

type Store struct {
	Products []Product
}

type Product struct {
	ID    string
	Name  string
	Price int64
}

func (s *Store) AddProduct(product Product) {
	s.Products = append(s.Products, product)
}

func (s *Store) GetProductByID(id string) (*Product, error) {
	for _, product := range s.Products {
		if product.ID == id {
			return &product, nil
		}
	}
	return nil, fmt.Errorf("product not found")
}

func (s *Store) ListProducts() []Product {
	return s.Products
}

func (s *Store) BuyProduct(productID string) error {
	product, err := s.GetProductByID(productID)
	if err != nil {
		return err
	}

	// Here you would integrate with a payment processor like Stripe
	// For example:
	err = stripe.BuyPremiumAccount(product.ID)
	if err != nil {
		return fmt.Errorf("failed to buy product: %w", err)
	}

	return nil
}
