package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln(err)
	}

	// Reading Minio configuration from environment variables
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("MINIO_SECRET_ACCESS_KEY")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"
	bucketName := os.Getenv("MINIO_BUCKET_NAME")

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Printf("error initializing Minio client")
		log.Fatalln(err)
	}

	r := gin.Default()

	// CORS middleware configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // Add your React app's URL
	r.Use(cors.New(config))

	r.POST("/upload", func(c *gin.Context) {
		// Get the file from the request
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
			return
		}

		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error opening the file"})
			return
		}
		defer src.Close()

		// Create a buffer to hold the contents of the uploaded file
		var buffer bytes.Buffer
		if _, err := io.Copy(&buffer, src); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error copying the file to buffer"})
			return
		}

		// Set a unique filename
		objectName := uuid.NewString() + filepath.Ext(file.Filename)

		// Upload the file to Minio
		info, err := minioClient.PutObject(context.Background(), bucketName, objectName, &buffer, int64(buffer.Len()), minio.PutObjectOptions{
			ContentType: file.Header.Get("Content-Type"),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error uploading the file to Minio"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully", "filename": info.Key})
	})

	r.GET("/images", func(c *gin.Context) {
		// Set request parameters for content-disposition.
		reqParams := make(url.Values)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{})

		var presignedURLs []string

		for object := range objectCh {
			if object.Err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": object.Err.Error()})
				return
			}

			// Generates a presigned url which expires in a day.
			presignedURL, err := minioClient.PresignedGetObject(context.Background(), bucketName, object.Key, time.Second*24*60*60, reqParams)
			if err != nil {
				fmt.Println(err)
				return
			}

			presignedURLs = append(presignedURLs, presignedURL.String())
		}

		c.JSON(http.StatusOK, presignedURLs)
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("error starting Gin server: %v", err)
	}
}
