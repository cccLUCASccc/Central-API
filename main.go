package main

import (
	"github.com/cccLUCASccc/Centra-API/routes"
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// Notre modèle basé sur ta capture d'écran


func main() {
	// Connexion à la DB via Railway
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err := sql.Open("postgres", dsn)

	api := &routes.Env{DB: db}

	if err != nil {
		log.Fatal(err)
	}

	// Définition de la première route
	http.HandleFunc("/api/vehicules", api.ListeVehicules)

	// Lancement du serveur sur le port fourni par Railway
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Serveur démarré sur le port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

