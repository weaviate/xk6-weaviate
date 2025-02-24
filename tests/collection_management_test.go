package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectionManagement(t *testing.T) {
	client := createTestClient(t)
	defer client.DeleteAllCollections()

	t.Run("create and delete collection", func(t *testing.T) {
		// Create a collection
		err := client.CreateCollection("TestCollection", map[string]interface{}{
			"description": "Test collection with text properties",
			"vectorizer":  "none",
			"properties": []map[string]interface{}{
				{
					"name":         "title",
					"description":  "Title of the document",
					"dataType":     []string{"text"},
					"tokenization": "word",
				},
				{
					"name":         "content",
					"description":  "Content of the document",
					"dataType":     []string{"text"},
					"tokenization": "word",
				},
			},
		})
		assert.NoError(t, err)

		// Delete the collection
		err = client.DeleteCollection("TestCollection")
		assert.NoError(t, err)
	})
}
