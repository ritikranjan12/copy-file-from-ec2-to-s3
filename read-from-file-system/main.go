package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

func writeToFile(filename string, content []string) error {
	var result []string
	for _, str := range content {
		splitParts := strings.Replace(str, "images\\", "", 1)
		result = append(result, splitParts)
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range result {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func uploadFolder(bucketName, folderPath, targetPath string, outputFilePath string) {
	folderList := []string{}
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(os.Getenv("awsDefaultRegion")),
		Credentials: credentials.NewStaticCredentials(os.Getenv("awsAccessKeyID"), os.Getenv("awsSecretAccessKey"), ""),
	})
	if err != nil {
		fmt.Println("Error creating session:", err)
		return
	}

	s3Client := s3.New(sess)
	cnt := 0
	err = filepath.Walk(folderPath, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && localPath != folderPath {
			folderList = append(folderList, localPath)
		}

		if !info.IsDir() {
			relativePath, err := filepath.Rel(folderPath, localPath)
			if err != nil {
				return err
			}

			s3Path := filepath.Join(targetPath, filepath.ToSlash(relativePath))
			s3Path = strings.Replace(s3Path, "\\", "/", -1)
			fmt.Printf("Uploading %s to %s\n", localPath, s3Path)

			file, err := os.Open(localPath)
			if err != nil {
				return err
			}
			defer file.Close()

			// Determine MIME type using file extension
			contentType := mime.TypeByExtension(filepath.Ext(localPath))

			_, err = s3Client.PutObject(&s3.PutObjectInput{
				Bucket:      aws.String(bucketName),
				Key:         aws.String(s3Path),
				Body:        file,
				ACL:         aws.String("public-read"),
				ContentType: aws.String(contentType),
			})
			if err != nil {
				fmt.Println("Error uploading file:", err)
				return err
			}
			cnt = cnt + 1
			fmt.Printf("Uploaded %s to s3://%s/%s\n", localPath, bucketName, s3Path)
			fmt.Printf("%d files uploaded\n", cnt)
		}

		err = writeToFile(outputFilePath, folderList)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error walking through the folder:", err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	bucketName := os.Getenv("bucketname")
	folderPath := "../images/"
	outputFilePath := "folder_list.txt"
	uploadFolder(bucketName, folderPath, "products/", outputFilePath)
}
