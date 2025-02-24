package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchOperations(t *testing.T) {
	client := createTestClient(t)
	defer client.DeleteAllCollections()

	t.Run("batch create and delete", func(t *testing.T) {
		// Create test collection
		err := client.CreateCollection("TestBatch", map[string]interface{}{
			"description": "Test collection for batch operations",
			"vectorizer":  "none",
			"properties": []map[string]interface{}{
				{
					"name":     "title",
					"dataType": []string{"text"},
				},
			},
		})
		require.NoError(t, err)

		// Batch create objects
		objects := []map[string]interface{}{
			{
				"class": "TestBatch",
				"properties": map[string]interface{}{
					"title": "Object 1",
				},
				"vector": []float32{0.1, 0.2, 0.3},
			},
			{
				"class": "TestBatch",
				"properties": map[string]interface{}{
					"title": "Object 2",
				},
				"vector": []float32{0.4, 0.5, 0.6},
			},
		}

		createResults, err := client.BatchCreate(objects)
		require.NoError(t, err)
		assert.Len(t, createResults, 2)
		for _, res := range createResults {
			assert.Equal(t, "success", res["status"])
		}

		// Batch delete objects
		deleteResponse, err := client.BatchDelete("TestBatch", map[string]interface{}{
			"where": map[string]interface{}{
				"operator":  "Like",
				"path":      []string{"title"},
				"valueText": "*",
			},
			"output":           "verbose",
			"consistencyLevel": "ONE",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleteResponse["successful"])
		assert.Equal(t, int64(0), deleteResponse["failed"])
		if objects, ok := deleteResponse["objects"].([]map[string]interface{}); ok {
			assert.Len(t, objects, 2)
			for _, obj := range objects {
				assert.Equal(t, "success", obj["status"])
			}
		} else {
			t.Fatal("objects field missing or invalid type")
		}

		// Cleanup
		err = client.DeleteCollection("TestBatch")
		assert.NoError(t, err)
	})
}
