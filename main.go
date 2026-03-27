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

    _ "github.com/lib/pq"
)

func main() {
    // 1. Connexion DB
    dsn := os.Getenv("DATABASE_URL")
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal("Erreur connexion DB:", err)
    }

    // 2. Configuration S3
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_DEFAULT_REGION")),
	)
	if err != nil {
		log.Fatal("Erreur config AWS:", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("AWS_ENDPOINT_URL"))
		o.UsePathStyle = true 
	})
    
    // 3. Initialisation de l'environnement des routes
    api := &routes.Env{
        DB:       db,
        S3Client: s3Client,
        Bucket:   os.Getenv("AWS_S3_BUCKET_NAME"),
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