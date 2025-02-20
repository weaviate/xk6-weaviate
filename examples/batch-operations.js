import weaviate from 'k6/x/weaviate';
import { check } from 'k6';

const client = weaviate.newClient({
  host: 'localhost:8080',
  scheme: 'http',
});

export default () => {
  // Create unique collection name using VU ID, iteration and timestamp
  const collectionName = `TestCollection_${__VU}_${__ITER}_${Date.now()}`;

  try {
    // Create test collection
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
      vector: [
        Math.random(),
        Math.random(), 
        Math.random()
      ]
    }));
    const createResults = client.batchCreate(objects);
    console.log("Batch Create Results:", JSON.stringify(createResults));

    // Batch delete objects
    const deleteResults = client.batchDelete(collectionName, {
      where: {
        operator: "Like",
        path: ["title"],
        valueText: "*"
      },
      output: "verbose"
    });
    console.log("Batch Delete Results:", JSON.stringify(deleteResults));
  } finally {
    // Cleanup - always try to delete the collection
    try {
      client.deleteCollection(collectionName);
    } catch (e) {
      console.error(`Failed to cleanup collection ${collectionName}: ${e.message}`);
    }
  }
}; 