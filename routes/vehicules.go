package routes

import (
	"github.com/cccLUCASccc/Centra-API/models"
	"encoding/json"
	"log"
	"net/http"
	"database/sql"
)

type Env struct {
    DB *sql.DB
}

func (e *Env) ListeVehicules(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")

	rows, err := e.DB.Query("SELECT id, model, description, price, sold, year FROM vehicules")
	if err != nil {
		http.Error(response, "Erreur lors de la lecture des données", 500)
		return
	}
	defer rows.Close()

	var catalogue []models.Vehicule

	for rows.Next() {
		var vehicule models.Vehicule
		err := rows.Scan(&vehicule.ID, &vehicule.Name, &vehicule.Description, &vehicule.Price, &vehicule.Sold, &vehicule.Year)
		if err != nil {
			log.Println("Erreur de scan :", err)
			continue
		}
		catalogue = append(catalogue, vehicule)
	}

	json.NewEncoder(response).Encode(catalogue)
}