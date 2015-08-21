package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

func downloadHTML(url string) (string, string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fmt.Println("Downloading", url, "to", fileName)

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return "", "", err
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return "", "", err
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return "", "", err
	}

	fmt.Println(n, "bytes downloaded.")

	return fileName, url, nil
}

func uploadImageToS3(path string, fileName string) error {
	fmt.Printf("Filename: %s\n", fileName)

	auth := aws.Auth{
		AccessKey: os.Getenv("ACCESS_KEY"),
		SecretKey: os.Getenv("SECRET_KEY"),
	}

	var region = aws.USEast

	client := s3.New(auth, region)

	data, err := ioutil.ReadFile(fileName)

	if err != nil {
		panic("error reading file! " + fileName)
	}

	bucket := client.Bucket("mesos-hackathon-bucket")
	options := s3.Options{}

	fmt.Printf("Path: %s\n", path)
	err = bucket.Put(path, data, "binary/octet-stream", s3.PublicRead, options)
	if err != nil {
		return err
	}

	return nil
}
