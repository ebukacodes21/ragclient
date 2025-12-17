package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Embedding struct {
	ID       string    `json:"id"`
	Values   []float32 `json:"values"`
	Metadata struct {
		Chunk  string `json:"chunk"`
		Source string `json:"source"`
	} `json:"metadata"`
}

func Upload(outputJSON []byte, collectionName string) {
	ctx := context.Background()

	var embeddings []Embedding
	if err := json.Unmarshal(outputJSON, &embeddings); err != nil {
		log.Fatalf("failed to parse embeddings: %v", err)
	}

	conn, err := grpc.NewClient("localhost:6334", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to Qdrant: %v", err)
	}
	defer conn.Close()

	client := qdrant.NewPointsClient(conn)

	createCollection(conn, ctx, collectionName)

	var points []*qdrant.PointStruct
	for _, e := range embeddings {
		payload := map[string]interface{}{
			"chunk":  e.Metadata.Chunk,
			"source": e.Metadata.Source,
		}

		points = append(points, &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{Uuid: e.ID},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: e.Values,
					},
				},
			},
			Payload: convertPayload(payload),
		})
	}

	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	})
	if err != nil {
		log.Fatalf("qdrant upsert failed: %v", err)
	}

	fmt.Printf("uploaded %d vectors to Qdrant collection [%s]\n", len(points), collectionName)
}

func createCollection(conn grpc.ClientConnInterface, ctx context.Context, name string) {
	collections := qdrant.NewCollectionsClient(conn)

	_, err := collections.Create(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     1536,
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("collection creation may have failed or already exists: %v\n", err)
	}
}

func convertPayload(data map[string]interface{}) map[string]*qdrant.Value {
	result := make(map[string]*qdrant.Value)
	for k, v := range data {
		switch val := v.(type) {
		case string:
			result[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
		}
	}
	return result
}
