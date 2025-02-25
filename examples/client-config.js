import weaviate from 'k6/x/weaviate';
import { check } from 'k6';

export default () => {
  // Test 1: Standard configuration
  const client1 = weaviate.newClient({
    host: 'localhost:8080',
    scheme: 'http',
    grpcHost: 'localhost:50051',
  });
  console.log('Client 1 created successfully');

  // Test 2: URL with scheme in host
  const client2 = weaviate.newClient({
    host: 'http://localhost:8080',
    grpcHost: 'localhost:50051',
  });
  console.log('Client 2 created successfully');

  // Test 3: HTTPS URL with scheme in host
  const client3 = weaviate.newClient({
    host: 'https://localhost:8080',
    grpcHost: 'localhost:50051',
  });
  console.log('Client 3 created successfully');

  // Test 4: Weaviate Cloud instance
  const client4 = weaviate.newClient({
    host: 'my-instance.c0.europe-west3.gcp.weaviate.cloud',
    // Note: grpcHost is not required for Weaviate Cloud instances
  });
  console.log('Client 4 created successfully');

  // Test 5: Weaviate Cloud instance with port
  const client5 = weaviate.newClient({
    host: 'my-instance.c0.europe-west3.gcp.weaviate.cloud:443',
    // Note: grpcHost is not required for Weaviate Cloud instances
  });
  console.log('Client 5 created successfully');

  // Test 6: Weaviate Cloud instance with scheme
  const client6 = weaviate.newClient({
    host: 'https://my-instance.c0.europe-west3.gcp.weaviate.cloud',
    // Note: grpcHost is not required for Weaviate Cloud instances
  });
  console.log('Client 6 created successfully');

  // Note: These clients won't actually connect to any server in this test
  // This is just to demonstrate that the client creation logic works
  console.log('All clients created successfully');
}; 