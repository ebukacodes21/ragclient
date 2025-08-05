package main

import (
	"myclient/container"
	"myclient/pinecone"
	"myclient/utils"

	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<30) // 10GB limit

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "error reading multipart data", http.StatusInternalServerError)
			return
		}

		if part.FormName() == "file" {
			safeFileName := utils.SanitizeFileName(filepath.Base(part.FileName()))
			uploadDir := filepath.Join(os.Getenv("HOME"), "uploads")
			_ = os.MkdirAll(uploadDir, 0755)
			tmpFilePath := filepath.Join(uploadDir, safeFileName)

			out, err := os.Create(tmpFilePath)
			if err != nil {
				http.Error(w, "unable to create file", http.StatusInternalServerError)
				return
			}
			defer out.Close()

			if _, err := io.Copy(out, part); err != nil {
				http.Error(w, "failed to save file", http.StatusInternalServerError)
				return
			}

			// run embedding container
			outputJSON, err := container.RunEmbeddingContainer(tmpFilePath)
			if err != nil {
				http.Error(w, fmt.Sprintf("embedding container failed: %v", err), http.StatusInternalServerError)
				return
			}

			// upload to Pinecone
			userId := "user123"
			pinecone.Upload(outputJSON, userId)

			// cleanup
			if err := os.Remove(tmpFilePath); err != nil {
				log.Printf("warning: failed to delete uploaded file: %v", err)
			}

			fileEntries, _ := os.ReadDir(uploadDir)
			for _, f := range fileEntries {
				fPath := filepath.Join(uploadDir, f.Name())
				if err := os.Remove(fPath); err != nil {
					log.Printf("warning: failed to delete file %s: %v", f.Name(), err)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("upload successful"))
			return
		}

		part.Close()
	}

	http.Error(w, "No file uploaded", http.StatusBadRequest)
}

func main() {
	http.HandleFunc("/upload", uploadHandler)
	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
