package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	endpoint := "192.168.255.112:9000"
	accessKeyID := "admin"
	secretAccessKey := "adminpass"
	useSSL := false

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

	r.GET("/images", func(c *gin.Context) {
		bucketName := "stable-diffusion"

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
