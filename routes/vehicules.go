package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cccLUCASccc/Centra-API/models"
)

type Env struct {
	DB       *sql.DB
	S3Client *s3.Client
	Bucket   string
}

// 1. LISTER LES VÉHICULES
func (e *Env) ListeVehicules(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")

	rows, err := e.DB.Query("SELECT id, model, description, price, sold, year, imageurl FROM vehicules")
	if err != nil {
		http.Error(response, "Erreur lors de la lecture des données", 500)
		return
	}
	defer rows.Close()

	var catalogue []models.Vehicule
	for rows.Next() {
		var v models.Vehicule
		err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.Price, &v.Sold, &v.Year, &v.ImageURL)
		if err != nil {
			log.Println("Erreur de scan :", err)
			continue
		}
		catalogue = append(catalogue, v)
	}
	json.NewEncoder(response).Encode(catalogue)
}

// 2. AJOUTER UN VÉHICULE + IMAGE
func (e *Env) AjouterVehicule(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	// A. Récupérer le fichier
	file, header, err := r.FormFile("image")
	var imageURL string
	if err == nil {
		defer file.Close()
		
		fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)

		// Upload vers S3
		_, err = e.S3Client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket: &e.Bucket,
			Key:    &fileName,
			Body:   file,
		})

		if err == nil {
			imageURL = fmt.Sprintf("%s/%s/%s", os.Getenv("AWS_ENDPOINT_URL"), e.Bucket, fileName)
		}
	}

	// B. Récupérer les données texte
	name := r.FormValue("name")
	desc := r.FormValue("description")

	pricestr := r.FormValue("price")
	pricefloat, err := strconv.ParseFloat(pricestr, 64)
	if err != nil {
		log.Println("Erreur conversion prix :", err)
		pricefloat = 0.0
	}
	price := float64(pricefloat)

	year := r.FormValue("year")

	// C. Insertion en DB
	query := `INSERT INTO vehicules (model, description, price, imagesurl, year) VALUES ($1, $2, $3, $4, $5)`
	_, err = e.DB.Exec(query, name, desc, price, imageURL, year)

	if err != nil {
		log.Printf("DÉTAIL ERREUR SQL : %v", err) 
		http.Error(w, fmt.Sprintf("Erreur SQL : %v", err), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
}