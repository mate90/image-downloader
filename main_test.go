package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDownloadImage(t *testing.T) {
	tempDir := t.TempDir()

	err := downloadImage("https://www.freecodecamp.org/news/content/images/size/w2000/2021/10/golang.png", tempDir, "test.jpg")
	if err != nil {
		t.Errorf("downloadImage failed: %v", err)
	}

	_, err = os.Stat(tempDir + "/test.jpg")
	if err != nil {
		t.Errorf("downloadImage did not create the file: %v", err)
	}
}

func TestResizeImage(t *testing.T) {
	inputImg := createSampleImage(t, 200, 200)

	tempDir := t.TempDir()

	inputPath := fmt.Sprintf("%s/input.jpg", tempDir)
	saveImageToFile(t, inputImg, inputPath)

	outputPath := fmt.Sprintf("%s/output.jpg", tempDir)
	err := resizeImage(inputPath, outputPath, 100, 100)
	if err != nil {
		t.Errorf("resizeImage failed: %v", err)
	}

	_, err = os.Stat(outputPath)
	if err != nil {
		t.Errorf("resizeImage did not create the resized image file: %v", err)
	}
}

func TestStoreImageInDatabase(t *testing.T) {
	resizedImg := createSampleImage(t, 100, 100)

	tempDir := t.TempDir()

	imagePath := fmt.Sprintf("%s/test.jpg", tempDir)
	saveImageToFile(t, resizedImg, imagePath)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database connection: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO images (.+) VALUES (.+)").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = storeImageInDatabase(db, "test.jpg", imagePath)
	if err != nil {
		t.Errorf("storeImageInDatabase failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Expectations were not met: %v", err)
	}
}

func createSampleImage(t *testing.T, width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	return img
}

func saveImageToFile(t *testing.T, img *image.NRGBA, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Error creating image file: %v", err)
	}
	defer file.Close()

	err = jpeg.Encode(file, img, nil)
	if err != nil {
		t.Fatalf("Error encoding and saving image: %v", err)
	}
}
