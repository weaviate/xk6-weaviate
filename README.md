# xk6-weaviate

K6 extension to interact with Weaviate vector database.

## Currently Supported Operations

### Collection Operations
- Create a collection with specified properties and configuration
- Delete a collection

### Object Operations
- Batch create objects with properties and vectors
- Batch delete objects based on where filters

### Multi-tenancy Operations
- Create tenants for a collection
- Update tenant status
- Delete tenants

## Build

To build a custom `k6` binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

1. Download [xk6](https://github.com/grafana/xk6):
    ```bash
    go install go.k6.io/xk6/cmd/xk6@latest
    ```

2. Build the k6 binary:
    ```bash
    xk6 build --with github.com/jfrancoa/xk6-weaviate
    ```

This will create a k6 binary that includes the xk6-weaviate extension in your local folder.

## Examples

### Prerequisites
- A Weaviate instance running locally on port 8080 (http://localhost:8080)

### Batch Operations Example
```javascript
import weaviate from 'k6/x/weaviate';
import { check } from 'k6';

const client = weaviate.newClient({
  host: 'localhost:8080',
  scheme: 'http',
});

export default () => {
  const collectionName = `TestCollection_${__VU}_${__ITER}_${Date.now()}`;

  try {
    // Create collection
    client.createCollection(collectionName, {
      description: "Test collection for batch operations",
      properties: [
        {
          name: "title",
          dataType: ["text"]
        }
      ]
    });

    // Batch create objects
    const objects = Array.from({ length: 1000 }, (_, i) => ({
      class: collectionName,
      properties: {
        title: `Document ${i + 1}`
      },
      vector: [Math.random(), Math.random(), Math.random()]
    }));
    client.batchCreate(objects);

    // Batch delete objects
    client.batchDelete(collectionName, {
      where: {
        operator: "Like",
        path: ["title"],
        valueText: "*"
      }
    });
  } finally {
    client.deleteCollection(collectionName);
  }
};
```

### Multi-tenancy Operations Example
```javascript
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
```

## Running Tests

1. Ensure you have a Weaviate instance running locally:
   ```bash
   docker run -d -p 8080:8080 semitechnologies/weaviate:latest
   ```

2. Run the examples using the built k6 binary:
   ```bash
   ./k6 run examples/batch-operations.js
   ./k6 run examples/tenant-operations.js
   ```
