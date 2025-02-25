package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate/entities/models"
)

func TestObjectInsert(t *testing.T) {
	client := createTestClient(t)
	defer client.DeleteAllCollections()

	t.Run("Basic object insertion", func(t *testing.T) {
		className := "TestInsertClass_" + time.Now().Format("20060102150405")
		// Create test class
		err := client.CreateCollection(className, map[string]interface{}{
			"properties": []map[string]interface{}{
				{
					"name":     "title",
					"dataType": []string{"text"},
				},
			},
		})
		require.Nil(t, err, "Collection creation failed with error: %v", err)
		obj := map[string]interface{}{
			"properties": map[string]interface{}{
				"title": "Test Document",
			},
			"vector": []interface{}{0.1, 0.2, 0.3},
		}

		result, err := client.ObjectInsert(className, obj)
		assert.NoError(t, err)
		assert.NotEmpty(t, result["id"])
		assert.Equal(t, "Test Document", result["properties"].(map[string]interface{})["title"])
		fetched, err := client.FetchObjects(className, map[string]interface{}{
			"id":         result["id"],
			"additional": []string{"vector"},
		})
		assert.NoError(t, err)
		assert.Len(t, fetched["objects"], 1)
		objects := fetched["objects"].([]map[string]interface{})
		assert.Equal(t, result["id"], objects[0]["id"])

		// Verify vector content
		vector := objects[0]["vector"].(models.C11yVector)
		expectedVector := []float32{0.1, 0.2, 0.3}
		assert.Equal(t, len(expectedVector), len(vector), "Vector length should match")
		for i := range expectedVector {
			assert.Equal(t, expectedVector[i], vector[i], "Vector element %d should match", i)
		}

		err = client.DeleteCollection(className)
		assert.NoError(t, err)
	})

	t.Run("Insert with custom ID", func(t *testing.T) {
		className := "TestInsertWithIDClass_" + time.Now().Format("20060102150405")
		// Create test class
		err := client.CreateCollection(className, map[string]interface{}{
			"properties": []map[string]interface{}{
				{
					"name":     "title",
					"dataType": []string{"text"},
				},
			},
		})
		require.Nil(t, err, "Collection creation failed with error: %v", err)
		customID := "123e4567-e89b-12d3-a456-426614174000"

		obj := map[string]interface{}{
			"id": customID,
			"properties": map[string]interface{}{
				"title": "Custom ID Document",
			},
		}

		result, err := client.ObjectInsert(className, obj)
		assert.NoError(t, err)
		assert.Equal(t, customID, result["id"])
		fetched, err := client.FetchObjects(className, map[string]interface{}{
			"id":         customID,
			"additional": []string{"vector"},
		})
		assert.NoError(t, err)
		assert.Len(t, fetched["objects"], 1)
		assert.Equal(t, "Custom ID Document", fetched["objects"].([]map[string]interface{})[0]["properties"].(map[string]interface{})["title"])
		err = client.DeleteCollection(className)
		assert.NoError(t, err)
	})

	t.Run("Insert with named vectors", func(t *testing.T) {
		className := "TestInsertNamedVector_" + time.Now().Format("20060102150405")

		err := client.CreateCollection(className, map[string]interface{}{
			"properties": []interface{}{
				map[string]interface{}{
					"name":     "title",
					"dataType": []interface{}{"text"},
				},
			},
			"vectorConfig": map[string]interface{}{
				"vector1": map[string]interface{}{
					"vectorizer": map[string]interface{}{
						"none": nil,
					},
					"vectorIndexType": "hnsw",
				},
				"vector2": map[string]interface{}{
					"vectorizer": map[string]interface{}{
						"none": nil,
					},
					"vectorIndexType": "flat",
					"vectorIndexConfig": map[string]interface{}{
						"hnsw": map[string]interface{}{
							"efConstruction": 128,
						},
					},
				},
			},
		})
		require.Nil(t, err)

		// Test insertion with named vectors
		obj := map[string]interface{}{
			"properties": map[string]interface{}{
				"title": "Vector Config Doc",
			},
			"vectors": map[string]interface{}{
				"vector1": []interface{}{0.1, 0.2, 0.3},
				"vector2": []interface{}{0.4, 0.5, 0.6},
			},
		}

		result, err := client.ObjectInsert(className, obj)
		assert.NoError(t, err)
		assert.Len(t, result["vectors"], 2)
		fetched, err := client.FetchObjects(className, map[string]interface{}{
			"id":         result["id"],
			"additional": []string{"vector"},
		})
		assert.NoError(t, err)

		// Just verify the vectors exist and the object was fetched
		objects := fetched["objects"].([]map[string]interface{})
		assert.Len(t, objects, 1)

		// Verify vectors content
		vectors := objects[0]["vectors"].(map[string]interface{})
		expectedVector1 := []float32{0.1, 0.2, 0.3}
		expectedVector2 := []float32{0.4, 0.5, 0.6}

		// Check vector lengths
		vector1 := vectors["vector1"].(models.Vector)
		vector2 := vectors["vector2"].(models.Vector)
		assert.Equal(t, len(expectedVector1), len(vector1), "Vector1 length should match")
		assert.Equal(t, len(expectedVector2), len(vector2), "Vector2 length should match")

		// Check vector contents
		for i := range expectedVector1 {
			assert.Equal(t, expectedVector1[i], vector1[i], "Vector1 element %d should match", i)
		}
		for i := range expectedVector2 {
			assert.Equal(t, expectedVector2[i], vector2[i], "Vector2 element %d should match", i)
		}

		err = client.DeleteCollection(className)
		assert.NoError(t, err)
	})

	t.Run("Insert with tenant", func(t *testing.T) {
		className := "TestInsertMTClass_" + time.Now().Format("20060102150405")
		// Create test class
		err := client.CreateCollection(className, map[string]interface{}{
			"properties": []map[string]interface{}{
				{
					"name":     "title",
					"dataType": []string{"text"},
				},
			},
			"multiTenancy": map[string]interface{}{
				"enabled": true,
			},
		})
		require.Nil(t, err, "Collection creation failed with error: %v", err)

		tenantName := "tenantA"
		err = client.CreateTenant(className, []map[string]interface{}{
			{"name": tenantName},
		})
		assert.NoError(t, err)

		obj := map[string]interface{}{
			"properties": map[string]interface{}{
				"title": "Tenant Document",
			},
			"tenant": tenantName,
		}

		result, err := client.ObjectInsert(className, obj)
		assert.NoError(t, err)
		assert.Equal(t, tenantName, result["tenant"])
		fetched, err := client.FetchObjects(className, map[string]interface{}{
			"id":         result["id"],
			"tenant":     tenantName,
			"additional": []string{"vector"},
		})
		assert.NoError(t, err)
		assert.Len(t, fetched["objects"], 1)
		err = client.DeleteTenant(className, []string{tenantName})
		assert.NoError(t, err)
		err = client.DeleteCollection(className)
		assert.NoError(t, err)
	})

	t.Run("Insert with consistency level", func(t *testing.T) {
		className := "TestInsertConsistencyClass_" + time.Now().Format("20060102150405")
		// Create test class
		err := client.CreateCollection(className, map[string]interface{}{
			"properties": []map[string]interface{}{
				{
					"name":     "title",
					"dataType": []string{"text"},
				},
			},
		})
		require.Nil(t, err, "Collection creation failed with error: %v", err)
		obj := map[string]interface{}{
			"properties": map[string]interface{}{
				"title": "Consistency Document",
			},
			"consistencyLevel": "quorum",
		}

		result, err := client.ObjectInsert(className, obj)
		assert.NoError(t, err)
		assert.NotEmpty(t, result["id"])
		err = client.DeleteCollection(className)
		assert.NoError(t, err)
	})

	t.Run("Invalid consistency level", func(t *testing.T) {
		className := "TestInsertConsistencyClass_" + time.Now().Format("20060102150405")
		// Create test class
		err := client.CreateCollection(className, map[string]interface{}{
			"properties": []map[string]interface{}{
				{
					"name":     "title",
					"dataType": []string{"text"},
				},
			},
		})
		require.Nil(t, err, "Collection creation failed with error: %v", err)
		obj := map[string]interface{}{
			"properties": map[string]interface{}{
				"title": "Bad Consistency Doc",
			},
			"consistencyLevel": "invalid",
		}

		_, err = client.ObjectInsert(className, obj)
		assert.Error(t, err)
		err = client.DeleteCollection(className)
		assert.NoError(t, err)
	})

	t.Run("Fetch objects with pagination", func(t *testing.T) {
		className := "TestFetchPagination_" + time.Now().Format("20060102150405")
		err := client.CreateCollection(className, map[string]interface{}{
			"properties": []map[string]interface{}{
				{"name": "index", "dataType": []string{"int"}},
			},
		})
		require.Nil(t, err)

		// Insert 5 test objects with ordered UUIDs
		for i := 0; i < 5; i++ {
			// Create UUID that ends with the index number to ensure order
			orderedID := fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i)
			obj := map[string]interface{}{
				"id": orderedID,
				"properties": map[string]interface{}{
					"index": i,
				},
			}
			result, err := client.ObjectInsert(className, obj)
			assert.NoError(t, err)
			assert.Equal(t, orderedID, result["id"], "ID should match the ordered UUID")
		}

		// Test limit only - should get first 2 objects
		fetched, err := client.FetchObjects(className, map[string]interface{}{
			"limit": 2,
		})
		assert.NoError(t, err)
		assert.Len(t, fetched["objects"], 2)
		objects := fetched["objects"].([]map[string]interface{})
		assert.Equal(t, float64(0), objects[0]["properties"].(map[string]interface{})["index"])
		assert.Equal(t, float64(1), objects[1]["properties"].(map[string]interface{})["index"])

		// Test offset - should get objects starting from index 2
		fetched, err = client.FetchObjects(className, map[string]interface{}{
			"offset": 2,
			"limit":  2,
		})
		assert.NoError(t, err)
		assert.Len(t, fetched["objects"], 2)
		objects = fetched["objects"].([]map[string]interface{})
		assert.Equal(t, float64(2), objects[0]["properties"].(map[string]interface{})["index"])
		assert.Equal(t, float64(3), objects[1]["properties"].(map[string]interface{})["index"])

		// Test last page - should get only the last object
		fetched, err = client.FetchObjects(className, map[string]interface{}{
			"offset": 4,
			"limit":  2,
		})
		assert.NoError(t, err)
		assert.Len(t, fetched["objects"], 1) // Should only get the last object
		objects = fetched["objects"].([]map[string]interface{})
		assert.Equal(t, float64(4), objects[0]["properties"].(map[string]interface{})["index"], "Last object should have index 4")

		err = client.DeleteCollection(className)
		assert.NoError(t, err)
	})
}
