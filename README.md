# xk6-weaviate

K6 extension to interact with Weaviate vector database.

## Currently Supported Operations

### Collection Operations
- Create a collection with specified properties and configuration
- Delete a collection

### Object Operations
- Batch create objects with properties and vectors
- Batch delete objects based on where filters
- Insert individual objects with properties and vectors
- Fetch objects with various filtering options

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

## Client Configuration

The client can be configured in several ways:

### Basic Configuration
```javascript
const client = weaviate.newClient({
  host: 'localhost:8080',
  scheme: 'http',
  grpcHost: 'localhost:50051',
});
```

### URL with Scheme
You can include the scheme in the host URL, and it will be automatically extracted:
```javascript
const client = weaviate.newClient({
  host: 'http://localhost:8080',
  grpcHost: 'localhost:50051',
});
```

### Weaviate Cloud Instances
For Weaviate Cloud instances, the client will automatically configure the appropriate settings:
```javascript
const client = weaviate.newClient({
  host: 'my-instance.c0.europe-west3.gcp.weaviate.cloud',
  // grpcHost is optional for Weaviate Cloud instances
});
```

The client will:
1. Set the scheme to 'https'
2. Add port 443 if not specified
3. Automatically generate the grpcHost by prepending 'grpc-' to the host

### Authentication
```javascript
// With API Key
const client = weaviate.newClient({
  host: 'localhost:8080',
  grpcHost: 'localhost:50051',
  apiKey: 'your-api-key',
});

// With Bearer Token
const client = weaviate.newClient({
  host: 'localhost:8080',
  grpcHost: 'localhost:50051',
  authToken: 'your-auth-token',
});
```

## Examples

### Prerequisites
- A Weaviate instance running locally on port 8080 (http://localhost:8080)

### Batch Operations Example
```javascript
import weaviate from 'k6/x/weaviate';
import { check } from 'k6';

const client = weaviate.newClient({
  host: 'np27xpxes6ybjbshntocgq.c0.europe-west3.gcp.weaviate.cloud',
  // For Weaviate Cloud, grpcHost is optional
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
  scheme: 'http',
  grpcHost: 'localhost:50051',
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
   docker run -d -p 8080:8080 -p 50051:50051 semitechnologies/weaviate:latest
   ```

2. Run the examples using the built k6 binary:
   ```bash
   ./k6 run examples/batch-operations.js
   ./k6 run examples/tenant-operations.js
   ./k6 run examples/object-operations.js
   ./k6 run examples/client-config.js
   ```
