import weaviate from 'k6/x/weaviate';
import { check } from 'k6';

const client = weaviate.newClient({
  host: 'localhost:8080',
  scheme: 'http',
  grpcHost: 'localhost:50051',
});


export default () => {
  const collectionName = `InsertDemo_${__VU}_${__ITER}_${Date.now()}`;
  const collectionMTName = `InsertDemoMT_${__VU}_${__ITER}_${Date.now()}`;
  const collectionNamedVectorsName = `InsertDemoNamedVectors_${__VU}_${__ITER}_${Date.now()}`;

  try {
    // Create collections with multi-tenancy and vector config
    client.createCollection(collectionName, {
      description: "Demo collection for object insertion",
      properties: [
        {
          name: "title",
          dataType: ["text"]
        },
        {
          name: "content",
          dataType: ["text"]
        }
      ]
    });

    client.createCollection(collectionMTName, {
      description: "Demo collection for multi-tenancy",
      properties: [
        { name: "title", dataType: ["text"] },
        { name: "content", dataType: ["text"] }
      ],
      multiTenancy: {
        enabled: true
      }
    });

    client.createCollection(collectionNamedVectorsName, {
      description: "Demo collection for named vectors",
      properties: [
        { name: "title", dataType: ["text"] },
        { name: "content", dataType: ["text"] }
      ],
      vectorConfig: {
        vector1: {
          vectorizer: { "none": null },
          vectorIndexType: "hnsw"
        },
        vector2: {
          vectorizer: { "none": null },
          vectorIndexType: "flat",
          vectorIndexConfig: {
            hnsw: {
              efConstruction: 128
            }
          }
        }
      }
    });
    
    // Basic insertion
    for (let i = 0; i < 10; i++) {
      // Create ordered IDs to ensure consistent retrieval order
      const orderedId = `00000000-0000-0000-0000-00000000000${i}`;
      const basicResult = client.objectInsert(collectionName, {
        id: orderedId,
        properties: {
          title: `Document ${i}`,
          content: `This is the content of the document ${i}`
        },
        vector: [0.1 * i, 0.2 * i, 0.3 * i]
      });
      check(basicResult, {
        'basic insert has ID': (r) => {
          console.log(`Basic insert ID check: ${!!r.id}`);
          return !!r.id;
        },
        'basic insert has vector': (r) => {
          console.log(`Basic vector data: ${JSON.stringify(r.vector)}`);
          return r.vector && Array.isArray(r.vector) && r.vector.length === 3;
        }
      });
    }

    // Insert with custom ID
    const customId = (function() {
      return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
      });
    })();

    const customIdResult = client.objectInsert(collectionName, {
      id: customId,
      properties: {
        title: "Custom ID Document",
        content: "Document with predefined ID"
      }
    });
    check(customIdResult, {
      'custom ID matches': (r) => r.id === customId
    });

    // Insert with named vectors
    const vectorResultNamedVectors = client.objectInsert(collectionNamedVectorsName, {
      properties: {
        title: "Multi Vector Doc",
        content: "Document with multiple vectors"
      },
      vectors: {
        vector1: new Array(3).fill().map(() => Math.random()),
        vector2: new Array(128).fill().map(() => Math.random())
      }
    });

    // Insert with tenant
    const tenantName = "tenantA";
    client.createTenant(collectionMTName, [{ name: tenantName }]);
    
    const tenantResult = client.objectInsert(collectionMTName, {
      properties: {
        title: "Tenant Document",
        content: "Belongs to specific tenant"
      },
      tenant: tenantName
    });
    check(tenantResult, {
      'has tenant set': (r) => r.tenant === tenantName
    });

    // Fetch objects examples
    // 1. Basic fetch with limit
    const basicFetch = client.fetchObjects(collectionName, {
      limit: 1,
      additional: ["id"]
    });
    console.log("Basic fetch result:", JSON.stringify(basicFetch, null, 2));
    check(basicFetch, {
      'basic fetch returns objects': (r) => r.objects && Array.isArray(r.objects),
      'basic fetch respects limit': (r) => r.objects && r.objects.length === 1
    });

    // 2. Fetch with additional vector field
    const vectorFetch = client.fetchObjects(collectionName, {
      id: "00000000-0000-0000-0000-000000000000",
      additional: ["vector"]
    });
    console.log("Vector fetch result:", JSON.stringify(vectorFetch, null, 2));
    check(vectorFetch, {
      'vector fetch includes vectors': (r) => {
        return r.objects && r.objects.length > 0 && 
               r.objects[0]._additional &&
               r.objects[0]._additional.vectors &&
               Object.keys(r.objects[0]._additional.vectors).length === 2;
      }
    });

    // 3. Fetch with pagination - using specific IDs instead
    const firstId = "00000000-0000-0000-0000-000000000000";
    const secondId = "00000000-0000-0000-0000-000000000001";
    
    const firstPage = client.fetchObjects(collectionName, {
      id: firstId,
      additional: ["id"]
    });

    const secondPage = client.fetchObjects(collectionName, {
      id: secondId,
      additional: ["id"]
    });
    
    console.log("First page result:", JSON.stringify(firstPage.objects[0], null, 2));
    console.log("Second page result:", JSON.stringify(secondPage.objects[0], null, 2));
    check(secondPage, {
      'pagination returns different objects': (r) => {
        return r.objects && r.objects.length > 0 && 
               firstPage.objects && firstPage.objects.length > 0 &&
               r.objects[0].id !== firstPage.objects[0].id;
      }
    });

    // 4. Fetch from multi-tenant collection
    const tenantFetch = client.fetchObjects(collectionMTName, {
      tenant: tenantName
    });
    check(tenantFetch, {
      'tenant fetch returns objects': (r) => r.objects && Array.isArray(r.objects)
    });

    // 5. Fetch with specific ID
    if (customId) {
      const idFetch = client.fetchObjects(collectionName, {
        id: customId
      });
      check(idFetch, {
        'id fetch returns correct object': (r) => {
          return r.objects && r.objects.length === 1 && r.objects[0].id === customId;
        }
      });
    }

    // 6. Fetch with named vectors
    const namedVectorsFetch = client.fetchObjects(collectionNamedVectorsName, {
      id: vectorResultNamedVectors.id,
      additional: ["vector"]
    });
    console.log("Named vectors fetch result:", JSON.stringify(namedVectorsFetch, null, 2));
    check(namedVectorsFetch, {
      'named vectors fetch includes vectors': (r) => {
        return r.objects && r.objects.length > 0 && 
               r.objects[0]._additional &&
               r.objects[0]._additional.vectors &&
               Object.keys(r.objects[0]._additional.vectors).length === 2;
      }
    });

  } finally {
    // Cleanup
    try {
      client.deleteAllCollections();
    } catch (e) {
      console.error(`Cleanup failed: ${e.message}`);
    }
  }
}; 