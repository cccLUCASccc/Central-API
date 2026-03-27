package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// Notre modèle basé sur ta capture d'écran
type Vehicule struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	Sold        bool   `json:"sold"`
	Year        int    `json:"year"`
}

var db *sql.DB

func main() {
	// Connexion à la DB via Railway
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	// Définition de la première route
	http.HandleFunc("/api/vehicules", listeVehicules)

	// Lancement du serveur sur le port fourni par Railway
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Serveur démarré sur le port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func listeVehicules(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, name, description, price, sold, year FROM Vehicules")
	if err != nil {
		http.Error(response, "Erreur lors de la lecture des données", 500)
		return
	}
	defer rows.Close()

	var catalogue []Vehicule

	for rows.Next() {
		var vehicule Vehicule
		err := rows.Scan(&vehicule.ID, &vehicule.Name, &vehicule.Description, &vehicule.Price, &vehicule.Sold, &vehicule.Year)
		if err != nil {
			log.Println("Erreur de scan :", err)
			continue
		}
		catalogue = append(catalogue, vehicule)
	}

	json.NewEncoder(response).Encode(catalogue)
}