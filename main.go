package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nfnt/resize"
)

type SearchInput struct {
	SearchQuery string `json:"SearchQuery"`
	MaxImages   int    `json:"MaxImages"`
}

type ImageInfo struct {
	URL       string
	FileName  string
	Directory string
}

type Config struct {
	DatabaseURL     string
	DownloadDir     string
	ResizedDir      string
	MaxImageWorkers int
	ResizeWidth     uint
	ResizeHeight    uint
}

func downloadImage(url, directory string, fileName string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Create the file in the specified directory
	filePath := filepath.Join(directory, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy the image data to the file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	log.Println("Downloaded:", url)
	return nil
}

func resizeImage(inputPath, outputPath string, width, height uint) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	resizedImg := resize.Resize(width, height, img, resize.Lanczos3)
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	err = jpeg.Encode(outputFile, resizedImg, nil)
	if err != nil {
		return err
	}

	log.Println("Resized and saved:", outputPath)
	return nil
}

func storeImageInDatabase(db *sql.DB, filename, imagePath string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read image data
	imageData, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// Insert image data into the database
	_, err = db.Exec("INSERT INTO images (filename, data) VALUES ($1, $2)", filename, imageData)
	return err
}

func downloadResizeAndStoreWorker(images <-chan ImageInfo, db *sql.DB, resizeWidth, resizeHeight uint, resizedDirectory string, wg *sync.WaitGroup) {
	defer wg.Done()
	for imgInfo := range images {
		// Download image
		err := downloadImage(imgInfo.URL, imgInfo.Directory, imgInfo.FileName)
		if err != nil {
			log.Println("Error downloading image:", err)
			continue
		}

		// Resizing image and saving it in the resized directory
		inputPath := filepath.Join(imgInfo.Directory, imgInfo.FileName)
		outputPath := filepath.Join(resizedDirectory, imgInfo.FileName)
		err = resizeImage(inputPath, outputPath, resizeWidth, resizeHeight)
		if err != nil {
			log.Println("Error resizing image:", err)
			continue
		}

		// Store resized image in the database
		err = storeImageInDatabase(db, imgInfo.FileName, outputPath)
		if err != nil {
			log.Println("Error storing image in the database:", err)
		}

		// Delete resized file after storing it in the database
		err = os.Remove(outputPath)
		if err != nil {
			log.Println("Error deleting resized file:", err)
		}

	}
}

func loadInputs(filename string) ([]SearchInput, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var inputs []SearchInput
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&inputs)
	if err != nil {
		return nil, err
	}

	return inputs, nil
}

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file:", err)
	}
	host := os.Getenv("DB_HOST")
	port, _ := strconv.ParseInt(os.Getenv("DB_PORT"), 10, 64)
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable", host, port, dbname, user, password)

	config := Config{
		DatabaseURL:     dsn,
		DownloadDir:     "./images",
		ResizedDir:      "./resized_images",
		MaxImageWorkers: 5,
		ResizeWidth:     100,
		ResizeHeight:    100,
	}

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	_, err = db.Exec(`
	    CREATE TABLE IF NOT EXISTS images (
	        id SERIAL PRIMARY KEY,
	        filename VARCHAR(255) NOT NULL,
	        data BYTEA NOT NULL
	    );
	`)
	if err != nil {
		log.Fatal("Error creating 'images' table:", err)
	}

	inputs, err := loadInputs("inputs.json")
	if err != nil {
		log.Fatal("Error loading search input:", err)
	}

	for _, input := range inputs {
		var wg sync.WaitGroup
		// searchQuery := input.SearchQuery
		searchQuery := strings.Replace(input.SearchQuery, " ", "_", -1)
		maxImages := input.MaxImages

		c := colly.NewCollector()

		var imageUrls []string

		c.OnHTML("img", func(e *colly.HTMLElement) {
			imageURL := e.Attr("src")
			if strings.HasPrefix(imageURL, "http") {
				imageUrls = append(imageUrls, imageURL)
			}
		})

		c.OnRequest(func(r *colly.Request) {
			log.Println("Visiting", r.URL)
		})

		c.OnError(func(r *colly.Response, err error) {
			log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
		})

		c.Visit(fmt.Sprintf("https://www.google.com/search?q=%s&tbm=isch&num=%d", searchQuery, maxImages))

		if err := os.MkdirAll(config.DownloadDir, os.ModePerm); err != nil {
			log.Fatal("Error creating download directory:", err)
		}

		if err := os.MkdirAll(config.ResizedDir, os.ModePerm); err != nil {
			log.Fatal("Error creating resized images directory:", err)
		}

		imageChannel := make(chan ImageInfo, len(imageUrls))

		wg.Add(config.MaxImageWorkers)
		for i := 0; i < config.MaxImageWorkers; i++ {
			go downloadResizeAndStoreWorker(imageChannel, db, config.ResizeWidth, config.ResizeHeight, config.ResizedDir, &wg)
		}

		for _, url := range imageUrls {
			fileName := strconv.FormatInt(rand.Int63(), 10) + ".jpg"
			imgInfo := ImageInfo{
				URL:       url,
				FileName:  fileName,
				Directory: config.DownloadDir,
			}
			imageChannel <- imgInfo
		}

		close(imageChannel)
		wg.Wait()
	}

}
