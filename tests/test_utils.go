package tests

import (
	"testing"

	"github.com/weaviate/xk6-weaviate"
)

func createTestClient(t *testing.T) *weaviate.Client {
	w := &weaviate.Weaviate{}
	client, err := w.NewClient(map[string]interface{}{
		"host":     "localhost:8080",
		"scheme":   "http",
		"grpcHost": "localhost:50051",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}
