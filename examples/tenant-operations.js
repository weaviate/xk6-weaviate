import weaviate from 'k6/x/weaviate';

const client = weaviate.newClient({
  host: 'localhost:8080',
  scheme: 'http'
});

export default () => {
  // Create multi-tenant collection
  client.createCollection("MultiTenantCollection", {
    description: "Collection with multi-tenancy enabled",
    multiTenancy: {
      enabled: true
    },
    properties: [
      {
        name: "name",
        dataType: ["text"]
      }
    ]
  });

  // Create tenants
  client.createTenant("MultiTenantCollection", [
    { name: "tenant1" },
    { name: "tenant2" }
  ]);

  // Update tenant status
  client.updateTenant("MultiTenantCollection", [
    {
      name: "tenant1",
      activityStatus: "COLD"
    }
  ]);

  // Delete tenants
  client.deleteTenant("MultiTenantCollection", ["tenant1", "tenant2"]);

  // Cleanup
  client.deleteCollection("MultiTenantCollection");
}; 