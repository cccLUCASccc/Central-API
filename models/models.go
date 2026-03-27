package models

type Vehicule struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       float64    `json:"price"`
	Sold        bool   `json:"sold"`
	Year        int    `json:"year"`
	Images    []string `json:"images"`
}