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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/exp/slices"
)

func main() {
	// Read server configuration from environment variables
	serverPort := ":" + os.Getenv("SERVER_PORT")

	// Read Minio configuration from environment variables
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("MINIO_SECRET_ACCESS_KEY")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"
	bucketName := os.Getenv("MINIO_BUCKET_NAME")

	splashScreen, err := os.ReadFile("splash_screen.txt")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(splashScreen))

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

	// Set trusted proxies
	r.ForwardedByClientIP = true
	r.SetTrustedProxies([]string{"127.0.0.1"})

	r.POST("/storage-service/upload", func(c *gin.Context) {
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

		supportedFileExtensions := []string{".png", ".jpg", ".jpeg", ".gif", ".mp4"}

		if !slices.Contains(supportedFileExtensions, filepath.Ext(file.Filename)) {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Unsupported file extension"})
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

	r.DELETE("/storage-service/delete/:image-id", func(c *gin.Context) {
		imageId := c.Param("image-id")

		err = minioClient.RemoveObject(context.Background(), bucketName, imageId, minio.RemoveObjectOptions{})
		if err != nil {
			fmt.Println(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "The file was removed from the server", "filename": imageId})
	})

	r.GET("/storage-service/images", func(c *gin.Context) {
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

	if err := r.Run(serverPort); err != nil {
		log.Fatalf("error starting Gin server: %v", err)
	}
}
