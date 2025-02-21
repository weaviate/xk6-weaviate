package weaviate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/data/replication"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/grpc"
	"github.com/weaviate/weaviate/entities/models"
	"go.k6.io/k6/js/modules"
)

// Weaviate represents the root client module
type Weaviate struct{}

// GetStringValue extracts a string value from a map
func GetStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetStringSlice converts an interface to a string slice
func GetStringSlice(val interface{}) []string {
	if slice, ok := val.([]interface{}); ok {
		result := make([]string, len(slice))
		for i, v := range slice {
			result[i] = v.(string)
		}
		return result
	}
	return nil
}

// Client represents a Weaviate client instance
type Client struct {
	client *weaviate.Client
}

func init() {
	modules.Register("k6/x/weaviate", new(Weaviate))
}

// NewClient creates a new Weaviate client instance
// cfg is a map of configuration options
// scheme is the scheme to use for the client (http or https)
// host is the host to use for the client (e.g. localhost:8080)
// grpcHost is the host to use for the gRPC client (e.g. localhost:50051)
// authToken is the authentication token to use for the client
// apiKey is the API key to use for the client
// headers is a map of additional headers to use for the client
// timeout is the timeout to use for the client
func (*Weaviate) NewClient(cfg map[string]interface{}) (*Client, error) {
	// Default to http if scheme not provided
	scheme := "http"
	if schemeVal, ok := cfg["scheme"].(string); ok {
		scheme = schemeVal
	}

	host, ok := cfg["host"].(string)
	if !ok {
		return nil, fmt.Errorf("host is required in config")
	}

	grpcHost, ok := cfg["grpcHost"].(string)
	if !ok {
		return nil, fmt.Errorf("grpcHost is required in config")
	}

	config := weaviate.Config{
		Host:   host,
		Scheme: scheme,
		GrpcConfig: &grpc.Config{
			Host: grpcHost,
		},
	}

	// Handle authentication if provided
	if authToken, ok := cfg["authToken"].(string); ok {
		config.AuthConfig = auth.BearerToken{
			AccessToken: authToken,
		}
	} else if apiKey, ok := cfg["apiKey"].(string); ok {
		config.AuthConfig = auth.ApiKey{
			Value: apiKey,
		}
	}

	// Handle additional headers if provided
	if headers, ok := cfg["headers"].(map[string]string); ok {
		config.Headers = headers
	}

	// Handle timeout if provided
	if timeout, ok := cfg["timeout"].(float64); ok {
		config.StartupTimeout = time.Duration(timeout) * time.Second
	}

	client, err := weaviate.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create weaviate client: %w", err)
	}

	return &Client{client: client}, nil
}

// CreateCollection creates a new collection in Weaviate
func (c *Client) CreateCollection(collectionName string, collectionConfig map[string]interface{}) error {
	collection := &models.Class{
		Class:       collectionName,
		Description: GetStringValue(collectionConfig, "description"),
		Properties:  make([]*models.Property, 0),
	}

	// Handle vectorizer configuration
	if vectorizer, ok := collectionConfig["vectorizer"].(string); ok {
		collection.Vectorizer = vectorizer
	}

	// Handle vector index type
	if vectorIndexType, ok := collectionConfig["vectorIndexType"].(string); ok {
		collection.VectorIndexType = vectorIndexType
	}

	// Handle vector index config
	if vectorIndexConfig, ok := collectionConfig["vectorIndexConfig"].(map[string]interface{}); ok {
		collection.VectorIndexConfig = vectorIndexConfig
	}

	// Handle inverted index config
	if invertedIndexConfig, ok := collectionConfig["invertedIndexConfig"].(map[string]interface{}); ok {
		collection.InvertedIndexConfig = &models.InvertedIndexConfig{}
		if bm25Config, ok := invertedIndexConfig["bm25"].(map[string]interface{}); ok {
			collection.InvertedIndexConfig.Bm25 = &models.BM25Config{
				K1: bm25Config["k1"].(float32),
				B:  bm25Config["b"].(float32),
			}
		}
		if stopwords, ok := invertedIndexConfig["stopwords"].(map[string]interface{}); ok {
			collection.InvertedIndexConfig.Stopwords = &models.StopwordConfig{
				Preset:    stopwords["preset"].(string),
				Additions: GetStringSlice(stopwords["additions"]),
				Removals:  GetStringSlice(stopwords["removals"]),
			}
		}
	}

	// Handle multi-tenancy config
	if multiTenancy, ok := collectionConfig["multiTenancy"].(map[string]interface{}); ok {
		collection.MultiTenancyConfig = &models.MultiTenancyConfig{
			Enabled: multiTenancy["enabled"].(bool),
		}
	}

	// Handle class properties
	if props, ok := collectionConfig["properties"].([]interface{}); ok {
		for _, p := range props {
			if propMap, ok := p.(map[string]interface{}); ok {
				property := &models.Property{
					Name:         propMap["name"].(string),
					Description:  GetStringValue(propMap, "description"),
					DataType:     GetStringSlice(propMap["dataType"]),
					Tokenization: GetStringValue(propMap, "tokenization"),
				}
				collection.Properties = append(collection.Properties, property)
			}
		}
	}

	return c.client.Schema().ClassCreator().
		WithClass(collection).
		Do(context.Background())
}

// DeleteCollection deletes a collection from Weaviate
func (c *Client) DeleteCollection(collectionName string) error {
	return c.client.Schema().
		ClassDeleter().
		WithClassName(collectionName).
		Do(context.Background())
}

// CreateTenant creates one or more tenants for a collection
func (c *Client) CreateTenant(collectionName string, tenants []map[string]interface{}) error {
	modelTenants := make([]models.Tenant, len(tenants))
	for i, t := range tenants {
		modelTenants[i] = models.Tenant{
			Name: GetStringValue(t, "name"),
		}
	}

	return c.client.Schema().
		TenantsCreator().
		WithClassName(collectionName).
		WithTenants(modelTenants...).
		Do(context.Background())
}

// DeleteTenant deletes one or more tenants from a collection
func (c *Client) DeleteTenant(collectionName string, tenantNames []string) error {
	return c.client.Schema().
		TenantsDeleter().
		WithClassName(collectionName).
		WithTenants(tenantNames...).
		Do(context.Background())
}

// UpdateTenant updates the status of one or more tenants
func (c *Client) UpdateTenant(collectionName string, tenants []map[string]interface{}) error {
	modelTenants := make([]models.Tenant, len(tenants))
	for i, t := range tenants {
		modelTenants[i] = models.Tenant{
			Name:           GetStringValue(t, "name"),
			ActivityStatus: GetStringValue(t, "activityStatus"),
		}
	}

	return c.client.Schema().
		TenantsUpdater().
		WithClassName(collectionName).
		WithTenants(modelTenants...).
		Do(context.Background())
}

// BatchCreate creates multiple objects in a batch operation
func (c *Client) BatchCreate(objects []map[string]interface{}) ([]map[string]interface{}, error) {
	modelObjects := make([]*models.Object, len(objects))
	for i, obj := range objects {
		className, ok := obj["class"].(string)
		if !ok {
			return nil, fmt.Errorf("object at index %d missing class name", i)
		}

		modelObj := &models.Object{
			Class: className,
		}

		// Handle ID if provided
		if id, ok := obj["id"].(string); ok {
			modelObj.ID = strfmt.UUID(id)
		}

		// Handle properties
		if props, ok := obj["properties"].(map[string]interface{}); ok {
			modelObj.Properties = props
		}

		// Handle vector if provided
		if vectors, ok := obj["vectors"].(map[string]interface{}); ok {
			modelObj.Vectors = make(models.Vectors, len(vectors))
			for name, vec := range vectors {
				if vector, ok := vec.([]float32); ok {
					modelObj.Vectors[name] = vector
				}
			}
		} else if vector, ok := obj["vector"].([]float32); ok {
			modelObj.Vector = vector
		}

		// Handle vector weights
		if weights, ok := obj["vectorWeights"].(map[string]float32); ok {
			modelObj.VectorWeights = weights
		}

		// Handle tenant
		if tenant, ok := obj["tenant"].(string); ok {
			modelObj.Tenant = tenant
		}

		modelObjects[i] = modelObj
	}

	results, err := c.client.Batch().
		ObjectsBatcher().
		WithObjects(modelObjects...).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	// Convert results to simplified map for JS
	output := make([]map[string]interface{}, len(results))
	for i, result := range results {
		res := map[string]interface{}{
			"class":  result.Class,
			"id":     result.ID.String(),
			"status": strings.ToLower(*result.Result.Status),
		}

		if result.Result != nil && result.Result.Errors != nil {
			res["status"] = "error"
			res["error"] = result.Result.Errors.Error
		}

		output[i] = res
	}

	return output, nil
}

// BatchDelete deletes multiple objects based on a where filter
func (c *Client) BatchDelete(className string, options map[string]interface{}) (map[string]interface{}, error) {
	batchDeleter := c.client.Batch().
		ObjectsBatchDeleter().
		WithClassName(className)

	// Handle where filter
	if whereFilter, ok := options["where"].(map[string]interface{}); ok {
		where := filters.Where()

		if operator, ok := whereFilter["operator"].(string); ok {
			switch operator {
			case "Equal":
				where.WithOperator(filters.Equal)
			case "Like":
				where.WithOperator(filters.Like)
			case "ContainsAny":
				where.WithOperator(filters.ContainsAny)
			case "LessThan":
				where.WithOperator(filters.LessThan)
			}
		}

		if path, ok := whereFilter["path"].([]string); ok {
			where = where.WithPath(path)
		} else if pathInterface, ok := whereFilter["path"].([]interface{}); ok {
			path := make([]string, len(pathInterface))
			for i, v := range pathInterface {
				path[i] = v.(string)
			}
			where = where.WithPath(path)
		}

		if valueString, ok := whereFilter["valueString"].(string); ok {
			where = where.WithValueString(valueString)
		}

		if valueText, ok := whereFilter["valueText"].([]interface{}); ok {
			texts := make([]string, len(valueText))
			for i, v := range valueText {
				texts[i] = v.(string)
			}
			where = where.WithValueText(texts...)
		} else if valueText, ok := whereFilter["valueText"].(string); ok {
			where = where.WithValueText(valueText)
		}

		batchDeleter.WithWhere(where)
	}

	// Handle dry run option
	if dryRun, ok := options["dryRun"].(bool); ok {
		batchDeleter.WithDryRun(dryRun)
	}

	// Handle output format
	if output, ok := options["output"].(string); ok {
		batchDeleter.WithOutput(output)
	}

	// Handle tenant
	if tenant, ok := options["tenant"].(string); ok {
		batchDeleter.WithTenant(tenant)
	}

	replicationMap := map[string]string{
		"all":    replication.ConsistencyLevel.ALL,
		"one":    replication.ConsistencyLevel.ONE,
		"quorum": replication.ConsistencyLevel.QUORUM,
	}

	// Handle consistency level
	if consistencyLevel, ok := options["consistencyLevel"].(string); ok {
		batchDeleter.WithConsistencyLevel(replicationMap[consistencyLevel])
	}

	response, err := batchDeleter.Do(context.Background())
	if err != nil {
		return nil, err
	}

	// Convert response to simplified map for JS
	output := map[string]interface{}{
		"matches":    response.Results.Matches,
		"successful": response.Results.Successful,
		"failed":     response.Results.Failed,
	}

	if response.Results.Objects != nil {
		objects := make([]map[string]interface{}, len(response.Results.Objects))
		for i, obj := range response.Results.Objects {
			objects[i] = map[string]interface{}{
				"id":     obj.ID,
				"status": strings.ToLower(*obj.Status),
			}
			if obj.Errors != nil {
				objects[i]["error"] = obj.Errors.Error
			}
		}
		output["objects"] = objects
	}

	return output, nil
}
