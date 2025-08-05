package pinecone

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/pinecone-io/go-pinecone/v4/pinecone"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type Embedding struct {
	ID       string    `json:"id"`
	Values   []float32 `json:"values"`
	Metadata struct {
		Chunk  string `json:"chunk"`
		Source string `json:"source"`
	} `json:"metadata"`
}

func Upload(outputJSON []byte, namespace string) {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Fatal("no .env file found")
	}

	apiKey := os.Getenv("PINECONE_API_KEY")
	if apiKey == "" {
		log.Fatal("PINECONE_API_KEY is not set")
	}

	var embeddings []Embedding
	if err := json.Unmarshal(outputJSON, &embeddings); err != nil {
		log.Fatalf("failed to parse embeddings output: %v", err)
	}

	// connect to Pinecone
	pc, err := pinecone.NewClient(pinecone.NewClientParams{ApiKey: apiKey})
	if err != nil {
		log.Fatalf("failed to create Pinecone client: %v", err)
	}

	indexes, err := pc.ListIndexes(ctx)
	if err != nil {
		log.Fatalf("failed to list indexes: %v", err)
	}

	var host string
	for _, idx := range indexes {
		if idx.Name == "test-one" {
			host = idx.Host
			break
		}
	}
	if host == "" {
		log.Fatal("index 'test-one' not found")
	}

	idx, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      host,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("failed to connect to index: %v", err)
	}

	// convert to pinecone vectors
	var vectors []*pinecone.Vector
	for _, emb := range embeddings {
		values := emb.Values
		metaMap := map[string]interface{}{
			"chunk":  emb.Metadata.Chunk,
			"source": emb.Metadata.Source,
		}
		metadataStruct, err := structpb.NewStruct(metaMap)
		if err != nil {
			log.Fatalf("failed to convert metadata: %v", err)
		}
		vectors = append(vectors, &pinecone.Vector{
			Id:       emb.ID,
			Values:   &values,
			Metadata: metadataStruct,
		})
	}

	// upsert in batches
	batchSize := 100
	for i := 0; i < len(vectors); i += batchSize {
		end := i + batchSize
		if end > len(vectors) {
			end = len(vectors)
		}

		count, err := idx.UpsertVectors(ctx, vectors[i:end])
		if err != nil {
			log.Fatalf("upsert failed: %v", err)
		}

		fmt.Printf("upserted %d vectors (%d to %d)\n", count, i, end-1)
	}

	fmt.Println("all embeddings uploaded and indexed!")
}
