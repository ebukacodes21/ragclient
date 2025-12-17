package api

import (
	"html/template"
	"log"
	"myclient/container"
	"myclient/utils"
	"myclient/vector"
	"net/http"

	"fmt"
	"io"

	"os"
	"path/filepath"
)

type Handler struct {
	embedder *container.Embedder
}

func NewHandler(embedder *container.Embedder) *Handler {
	return &Handler{
		embedder: embedder,
	}
}

func (rh *Handler) Home(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./views/index.html"))
	tmpl.Execute(w, nil)
}

func (rh *Handler) UploadHandler(w http.ResponseWriter, r *http.Request) {
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
			outputJSON, err := rh.embedder.RunEmbedder(tmpFilePath)
			if err != nil {
				http.Error(w, fmt.Sprintf("embedding container failed: %v", err), http.StatusInternalServerError)
				return
			}

			// upload to Pinecone
			userId := "user123"
			vector.Upload(outputJSON, userId)

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
