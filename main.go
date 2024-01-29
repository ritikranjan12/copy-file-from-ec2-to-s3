package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Persons struct {
	Id               int    `json:"id"`
	Listing_id       int    `json:"listing_id"`
	Image_file_name  string `json:"image_file_name"`
	Output_file_name string
	Input_file_name  string
}

func writeToFile(filename string, content []string) error {
	var result []string
	for _, str := range content {
		splitParts := strings.Replace(str, "./images//", "", 1)
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
	jsonFilePath := "listing_images.json"

	// Read the content of the JSON file
	jsonData, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		log.Fatalf("Error reading JSON file: %v", err)
	}

	// Define a slice to unmarshal the JSON data into
	var persons []Persons
	// Unmarshal the JSON data into the slice
	err = json.Unmarshal(jsonData, &persons)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON data: %v", err)
	}
	s3Client := s3.New(sess)
	cnt := 0

	for i := range persons {
		persons[i].Input_file_name = strconv.Itoa(persons[i].Id) + "/original/" + persons[i].Image_file_name
		persons[i].Output_file_name = "products/" + strconv.Itoa(persons[i].Listing_id) + "/" + strconv.Itoa(persons[i].Id) + "-" + persons[i].Image_file_name
		folderPath1 := folderPath + "/" + strconv.Itoa(persons[i].Id) + "/original/"

		err = filepath.Walk(folderPath1, func(localPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && localPath != folderPath {
				folderList = append(folderList, localPath)
			}

			if !info.IsDir() {
				file, err := os.Open(localPath)
				if err != nil {
					return err
				}
				defer file.Close()

				// Determine MIME type using file extension
				contentType := mime.TypeByExtension(filepath.Ext(localPath))

				_, err = s3Client.PutObject(&s3.PutObjectInput{
					Bucket:      aws.String(bucketName),
					Key:         aws.String(persons[i].Output_file_name),
					Body:        file,
					ACL:         aws.String("public-read"),
					ContentType: aws.String(contentType),
				})
				if err != nil {
					fmt.Println("Error uploading file:", err)
					return err
				}
				cnt = cnt + 1
				fmt.Printf("Uploaded %s to s3://%s/%s\n", localPath, bucketName, persons[i].Output_file_name)
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
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	bucketName := os.Getenv("bucketname")
	folderPath := "./images"
	outputFilePath := "uploaded-image-ids.txt"
	uploadFolder(bucketName, folderPath, "products/", outputFilePath)
}
