import weaviate from 'k6/x/weaviate';
import { check } from 'k6';

const client = weaviate.newClient({
  host: 'localhost:8080',
  scheme: 'http',
  grpcHost: 'localhost:50051',
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
        },
        {
          name: "testId",
          dataType: ["text"]
        },
        {
          name: "timestamp",
          dataType: ["text"]
        }
      ]
    });

    // Generate a 30-dimensional vector similar to the user's format
    const generateLongVector = () => {
      const vector = [];
      for (let i = 0; i < 30; i++) {
        // Generate values between -1 and 1 to mimic the user's vector format
        vector.push((Math.random() * 2) - 1);
      }
      return vector;
    };

    // Create a small batch for testing (10 objects instead of 1000)
    const objects = Array.from({ length: 10 }, (_, i) => ({
      class: collectionName,
      properties: {
        title: `Document ${i + 1}`,
        testId: `test-${i}`,
        timestamp: new Date().toISOString()
      },
      vector: generateLongVector()
    }));

    console.log(`Creating ${objects.length} objects with 30-dimensional vectors`);
    
    // Log the first object to verify the vector format
    console.log("First object example:", JSON.stringify({
      class: objects[0].class,
      properties: objects[0].properties,
      vectorLength: objects[0].vector.length
    }));
    
    // Batch create objects
    const createResults = client.batchCreate(objects);
    console.log("Batch Create Results:", JSON.stringify(createResults));

    // Verify vectors were stored correctly by retrieving one object
    if (createResults && createResults.length > 0) {
      const firstObjectId = createResults[0].id;
      
      // Get the object to check if vector was stored
      const fetchResult = client.fetchObjects(collectionName, {
        id: firstObjectId,
        additional: ["vector"]
      });
      
      if (fetchResult && fetchResult.objects && fetchResult.objects.length > 0) {
        const retrievedObject = fetchResult.objects[0];
        
        // Check if vector exists and has the correct dimension
        const vectorExists = check(retrievedObject, {
          'Vector exists': (obj) => obj && obj.vector && Array.isArray(obj.vector),
          'Vector has correct dimension': (obj) => obj && obj.vector && obj.vector.length === 30
        });
        
        if (vectorExists) {
          console.log("Vector was successfully stored with length:", retrievedObject.vector.length);
          console.log("First few vector values:", retrievedObject.vector.slice(0, 5));
        } else {
          console.error("Vector was not stored correctly:", JSON.stringify(retrievedObject));
        }
      } else {
        console.error("Failed to fetch object:", JSON.stringify(fetchResult));
      }
      
      // Also try to fetch all objects to see if any have vectors
      const allObjects = client.fetchObjects(collectionName, {
        limit: 5,
        additional: ["vector"]
      });
      
      console.log("Sample of all objects:", JSON.stringify(allObjects));
    }

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