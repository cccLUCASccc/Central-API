package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// 1. LISTER LES projets
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

	listeDeProjets := []Projet{}

	for rows.Next() {
		var p Projet
		// Utilise pq.Array pour le tableau d'images
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