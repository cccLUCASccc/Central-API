package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cccLUCASccc/Centra-API/routes"
	"github.com/aws/aws-sdk-go-v2/credentials"

	_ "github.com/lib/pq"
)

func main() {
	// 1. Connexion à la Base de Données
	dsn := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Erreur ouverture DB:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Impossible de joindre la DB:", err)
	}

	// 2. Lire les variables S3
	bucketName := os.Getenv("AWS_S3_BUCKET_NAME")
	endpointURL := os.Getenv("AWS_ENDPOINT_URL")
	region := os.Getenv("AWS_DEFAULT_REGION")

	creds := credentials.NewStaticCredentialsProvider(
        os.Getenv("AWS_ACCESS_KEY_ID"),
        os.Getenv("AWS_SECRET_ACCESS_KEY"),
        "",
    )

	log.Printf("DÉMARRAGE - Bucket: %s | Endpoint: %s", bucketName, endpointURL)

	if bucketName == "" || endpointURL == "" {
		log.Fatal("ERREUR: Variables S3 (Bucket ou Endpoint) manquantes dans Railway !")
	}

	// 3. Créer la config AWS
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		log.Fatal("Erreur config AWS:", err)
	}

	// 4. Créer le client S3
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpointURL)
		o.UsePathStyle = true
	})

	// 5. Passer les valeurs à l'environnement des routes
	api := &routes.Env{
		DB:       db,
		S3Client: s3Client,
		Bucket:   bucketName,
	}

	// Routes
	http.HandleFunc("/api/vehicules", api.ListeVehicules)
	http.HandleFunc("/api/vehicules/add", api.AjouterVehicule)

	// Lancement
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Serveur démarré sur le port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}