package main

import (
	"log"
	"myclient/api"
	"myclient/container"
	"net/http"
)

var (
	embedder *container.Embedder
	handler  *api.Handler
)

func init() {
	embedder = container.NewEmbedder()
	handler = api.NewHandler(embedder)
}

func main() {

	http.HandleFunc("/upload", handler.UploadHandler)
	http.HandleFunc("/", handler.Home)

	log.Println("Listening on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
