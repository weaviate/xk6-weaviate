package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantManagement(t *testing.T) {
	client := createTestClient(t)

	// Create a collection for tenant testing
	err := client.CreateCollection("MultiTenantCollection", map[string]interface{}{
		"description": "Multi-tenant collection",
		"multiTenancy": map[string]interface{}{
			"enabled": true,
		},
		"properties": []map[string]interface{}{
			{
				"name":     "name",
				"dataType": []string{"text"},
			},
		},
	})
	assert.NoError(t, err)

	t.Run("manage tenants", func(t *testing.T) {
		// Create tenants
		err := client.CreateTenant("MultiTenantCollection", []map[string]interface{}{
			{"name": "tenant1"},
			{"name": "tenant2"},
		})
		assert.NoError(t, err)

		// Update tenant status
		err = client.UpdateTenant("MultiTenantCollection", []map[string]interface{}{
			{
				"name":           "tenant1",
				"activityStatus": "COLD",
			},
		})
		assert.NoError(t, err)

		// Delete tenants
		err = client.DeleteTenant("MultiTenantCollection", []string{"tenant1", "tenant2"})
		assert.NoError(t, err)
	})

	// Cleanup
	err = client.DeleteCollection("MultiTenantCollection")
	assert.NoError(t, err)
}
