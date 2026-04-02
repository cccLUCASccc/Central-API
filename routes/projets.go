package projets

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

// 1. LISTER LES projets
func (e *Env) ListeProjets(w http.ResponseWriter, r *http.Request) {
    query := `
        SELECT p.id, p.name, p.description, p.status, 
               COALESCE(array_agg(pi.url) FILTER (WHERE pi.url IS NOT NULL), '{}')
        FROM projets p
        LEFT JOIN projet_images pi ON p.id = pi.vehicule_id
        GROUP BY p.id`

    rows, err := e.DB.Query(query)
    if err != nil {
        log.Printf("Erreur Query: %p", err)
        http.Error(w, "Erreur lecture DB", 500)
        return
    }
    defer rows.Close()

    var Liste_De_Projets []models.Projet
    
    for rows.Next() {
        var v models.Projet
        err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.Image)
        if err != nil {
            log.Printf("Erreur Scan: %p", err)
            continue
        }
        Liste_De_Projets = append(Liste_De_Projets, p)
    }

    presignClient := s3.NewPresignClient(e.S3Client)

    for i := range Liste_De_Projets {
        for j, url := range Liste_De_Projets[i].Images {
            key := extractKeyFromURL(url) 

            presignedReq, err := presignClient.PresignGetObject(r.Context(), &s3.GetObjectInput{
                Bucket: &e.Bucket,
                Key:    &key,
            }, s3.WithPresignExpires(24 * time.Hour))

            if err == nil {
                Liste_De_Projets[i].Images[j] = presignedReq.URL
            }
        }
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(Liste_De_Projets)
}

func extractKeyFromURL(fullURL string) string {
    parts := strings.Split(fullURL, "/")
    return parts[len(parts)-1]
}

// 2. AJOUTER UN Projet + IMAGE
func (e *Env) AjouterProjet(w http.ResponseWriter, r *http.Request) {
    // 1. Parser le formulaire (10 Mo max)
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Fichier trop lourd", 400)
        return
    }

    // 2. Récupérer les données texte
    name := r.FormValue("name")
    desc := r.FormValue("description")
    status := r.FormValue("status")

    // 3. Récupérer L'UNIQUE image
    file, header, err := r.FormFile("image")
    if err != nil {
        http.Error(w, "Image manquante ou invalide", 400)
        return
    }
    defer file.Close()

    // 4. Insérer le projet en base de données
    var projetID int
    queryProjet := `INSERT INTO projets (name, description, status) 
                    VALUES ($1, $2, $3) RETURNING id`
    
    err = e.DB.QueryRow(queryProjet, name, desc, status).Scan(&projetID)
    if err != nil {
        log.Printf("Erreur SQL Projet: %v", err)
        http.Error(w, "Erreur lors de la création du projet", 500)
        return
    }

    // 5. Préparer l'upload S3
    fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)
    
    _, err = e.S3Client.PutObject(r.Context(), &s3.PutObjectInput{
        Bucket:      &e.Bucket,
        Key:         &fileName,
        Body:        file,
        ACL:         "public-read",
        ContentType: aws.String(header.Header.Get("Content-Type")),
    })

    if err != nil {
        log.Printf("Erreur S3: %v", err)
        http.Error(w, "Erreur lors de l'upload de l'image", 500)
        return
    }

    // 6. Enregistrer l'URL de l'image
    // Note : On utilise l'endpoint T3 pour reconstruire l'URL
    imageURL := fmt.Sprintf("https://t3.storageapi.dev/%s/%s", e.Bucket, fileName)
    
    _, err = e.DB.Exec("INSERT INTO projet_images (projet_id, url) VALUES ($1, $2)", projetID, imageURL)
    if err != nil {
        log.Printf("Erreur SQL Image: %v", err)
        // On ne coupe pas ici car le projet est déjà créé, mais c'est à surveiller
    }

    // 7. Réponse succès
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Projet ajouté avec succès !",
        "id":      projetID,
    })
}