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
	"github.com/lib/pq"
)

type Env struct {
	DB       *sql.DB
	S3Client *s3.Client
	Bucket   string
}

// 1. LISTER LES VÉHICULES
func (e *Env) ListeVehicules(w http.ResponseWriter, r *http.Request) {
    query := `
        SELECT v.id, v.model, v.description, v.price, v.sold, v.year, 
               COALESCE(array_agg(vi.url) FILTER (WHERE vi.url IS NOT NULL), '{}')
        FROM vehicules v
        LEFT JOIN vehicule_images vi ON v.id = vi.vehicule_id
        GROUP BY v.id`

    rows, err := e.DB.Query(query)
    if err != nil {
        log.Printf("Erreur Query: %v", err)
        http.Error(w, "Erreur lecture DB", 500)
        return
    }
    defer rows.Close()

    var catalogue []models.Vehicule
    for rows.Next() {
        var v models.Vehicule
        err := rows.Scan(&v.ID, &v.Model, &v.Description, &v.Price, &v.Sold, &v.Year, pq.Array(&v.Images))
        if err != nil {
            log.Printf("Erreur Scan: %v", err)
            continue
        }
        catalogue = append(catalogue, v)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(catalogue)
}

// 2. AJOUTER UN VÉHICULE + IMAGE
func (e *Env) AjouterVehicule(w http.ResponseWriter, r *http.Request) {
    r.ParseMultipartForm(10 << 20)

    // 2. Récupérer les données texte
    model := r.FormValue("model")
    desc := r.FormValue("description")
    price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
    year := r.FormValue("year")

    var vehiculeID int
    queryVehicule := `INSERT INTO vehicules (model, description, price, year) 
                      VALUES ($1, $2, $3, $4) RETURNING id`
    
    err := e.DB.QueryRow(queryVehicule, model, desc, price, year).Scan(&vehiculeID)
    if err != nil {
        log.Printf("Erreur SQL Vehicule: %v", err)
        http.Error(w, "Erreur insertion véhicule", 500)
        return
    }

    // 4. RÉCUPÉRER TOUTES LES IMAGES
    files := r.MultipartForm.File["image"]
    endpoint := os.Getenv("AWS_ENDPOINT_URL")

    for _, header := range files {
        file, err := header.Open()
        if err != nil {
            continue
        }

        fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)

        // Upload vers S3
        _, err = e.S3Client.PutObject(r.Context(), &s3.PutObjectInput{
            Bucket: &e.Bucket,
            Key:    &fileName,
            Body:   file,
        })
        file.Close()

        if err == nil {
            imageURL := fmt.Sprintf("%s/%s/%s", endpoint, e.Bucket, fileName)
            _, err = e.DB.Exec("INSERT INTO vehicule_images (vehicule_id, url) VALUES ($1, $2)", vehiculeID, imageURL)
            if err != nil {
                log.Printf("Erreur SQL Image: %v", err)
            }
        }
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"message": "Véhicule et images ajoutés !"})
}