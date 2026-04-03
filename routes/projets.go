package routes

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
	"strconv"

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

// 1. LISTER LES PROJETS
func (e *Env) ListeProjets(w http.ResponseWriter, r *http.Request) {
    query := `
        SELECT p.id, p.name, p.description, p.status, 
               COALESCE(array_agg(pi.url) FILTER (WHERE pi.url IS NOT NULL), '{}')
        FROM projets p
        LEFT JOIN projet_images pi ON p.id = pi.projet_id
        GROUP BY p.id`

    rows, err := e.DB.Query(query)
    if err != nil {
        log.Printf("Erreur Query ListeProjets: %v", err)
        http.Error(w, "Erreur lecture DB", 500)
        return
    }
    defer rows.Close()

    var liste_de_projets []Projet
    
    for rows.Next() {
        var p Projet
        // Le scan doit correspondre exactement au SELECT ci-dessus
        err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, pq.Array(&p.Images))
        if err != nil {
            log.Printf("Erreur Scan Projet: %v", err)
            continue
        }
        liste_de_projets = append(liste_de_projets, p)
    }

    // Gestion des URLs présignées (Optionnel si tes images sont publiques)
    presignClient := s3.NewPresignClient(e.S3Client)
    for i := range liste_de_projets {
        for j, url := range liste_de_projets[i].Images {
            key := extractKeyFromURL(url) 
            presignedReq, err := presignClient.PresignGetObject(r.Context(), &s3.GetObjectInput{
                Bucket: &e.Bucket,
                Key:    &key,
            }, s3.WithPresignExpires(24 * time.Hour))

            if err == nil {
                liste_de_projets[i].Images[j] = presignedReq.URL
            }
        }
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(liste_de_projets)
}

// 2. AJOUTER UN PROJET
func (e *Env) AjouterProjet(w http.ResponseWriter, r *http.Request) {
    r.ParseMultipartForm(10 << 20)

    name := r.FormValue("name")
    desc := r.FormValue("description")

    statusStr := r.FormValue("status")
	status, _ := strconv.Atoi(statusStr)

    var projetID int
    // On s'assure que la table s'appelle 'projets' et les colonnes 'name', 'description', 'status'
    queryProjet := `INSERT INTO projets (name, description, status) 
                    VALUES ($1, $2, $3) RETURNING id`

    err := e.DB.QueryRow(queryProjet, name, desc, status).Scan(&projetID)
    if err != nil {
        log.Printf("Erreur SQL Projet (INSERT): %v", err)
        http.Error(w, "Erreur insertion projet", 500)
        return
    }

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
            ACL:    "public-read",
        })
        file.Close()

        if err == nil {
            imageURL := fmt.Sprintf("%s/%s/%s", endpoint, e.Bucket, fileName)
            // Correction ici : projet_images au lieu de projets-images
            _, err = e.DB.Exec("INSERT INTO projet_images (projet_id, url) VALUES ($1, $2)", projetID, imageURL)
            if err != nil {
                log.Printf("Erreur SQL Image: %v", err)
            }
        }
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"message": "Projet ajouté !"})
}