package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lib/pq"
)

type Projet struct {
    ID          int      `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Images      []string `json:"images"` 
    Status      string   `json:"status"`
}

// 1. LISTER LES projets (Corrigé pour correspondre à la logique véhicules)
func (e *Env) ListeProjets(w http.ResponseWriter, r *http.Request) {
	query := `
        SELECT p.id, p.name, p.description, p.status, 
               COALESCE(array_agg(pi.url) FILTER (WHERE pi.url IS NOT NULL), '{}')
        FROM projets p
        LEFT JOIN projet_images pi ON p.id = pi.projet_id
        GROUP BY p.id`

	rows, err := e.DB.Query(query)
	if err != nil {
		log.Printf("Erreur Query: %v", err)
		http.Error(w, "Erreur lecture DB", 500)
		return
	}
	defer rows.Close()

	var listeDeProjets []Projet
	for rows.Next() {
		var p Projet
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, pq.Array(&p.Images))
		if err != nil {
			log.Printf("Erreur Scan: %v", err)
			continue
		}
		listeDeProjets = append(listeDeProjets, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listeDeProjets)
}

// 2. AJOUTER UN Projet (Copié de la logique AjouterVehicule qui fonctionne)
func (e *Env) AjouterProjet(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	name := r.FormValue("name")
	desc := r.FormValue("description")
	status := r.FormValue("status")

	var projetID int
	queryProjet := `INSERT INTO projets (name, description, status) 
                    VALUES ($1, $2, $3) RETURNING id`

	err := e.DB.QueryRow(queryProjet, name, desc, status).Scan(&projetID)
	if err != nil {
		log.Printf("Erreur SQL Projet: %v", err)
		http.Error(w, "Erreur insertion projet", 500)
		return
	}

	// On utilise la même logique de boucle que pour les véhicules
	files := r.MultipartForm.File["image"]
	endpoint := os.Getenv("AWS_ENDPOINT_URL")

	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			continue
		}

		fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)

		_, err = e.S3Client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket: &e.Bucket,
			Key:    &fileName,
			Body:   file,
			ACL:    "public-read", // On remet l'ACL qui fonctionne sur les véhicules
		})
		file.Close()

		if err == nil {
			imageURL := fmt.Sprintf("%s/%s/%s", endpoint, e.Bucket, fileName)
			_, err = e.DB.Exec("INSERT INTO projet_images (projet_id, url) VALUES ($1, $2)", projetID, imageURL)
			if err != nil {
				log.Printf("Erreur SQL Image: %v", err)
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Projet ajouté !"})
}

// 2. AJOUTER UN Projet
func (e *Env) AjouterProjet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Fichier trop lourd", 400)
		return
	}

	name := r.FormValue("name")
	desc := r.FormValue("description")
	status := r.FormValue("status")

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image manquante", 400)
		return
	}
	defer file.Close()

	var projetID int
	queryProjet := `INSERT INTO projets (name, description, status) 
                    VALUES ($1, $2, $3) RETURNING id`

	err = e.DB.QueryRow(queryProjet, name, desc, status).Scan(&projetID)
	if err != nil {
		log.Printf("Erreur SQL: %v", err)
		http.Error(w, "Erreur insertion", 500)
		return
	}

	fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)

	_, err = e.S3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &e.Bucket,
		Key:         &fileName,
		Body:        file,
		ACL:    "public-read",
	})

	if err == nil {
		imageURL := fmt.Sprintf("https://t3.storageapi.dev/%s/%s", e.Bucket, fileName)
		e.DB.Exec("INSERT INTO projets_images (projet_id, url) VALUES ($1, $2)", projetID, imageURL)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}