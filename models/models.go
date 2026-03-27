package models

type Vehicule struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	Sold        bool   `json:"sold"`
	Year        int    `json:"year"`
	ImageURL    string `json:"imageurl"`
}