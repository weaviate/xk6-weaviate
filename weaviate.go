package weaviate

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
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

// Add helper function at top of file with other helpers
func GetBoolValue(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}

// ToInt handles all numeric types from JS/Go conversions
func ToInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed, true
		}
		return 0, false
	default:
		// Handle other numeric types that might come from JS
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(rv.Int()), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(rv.Uint()), true
		case reflect.Float32, reflect.Float64:
			return int(rv.Float()), true
		default:
			return 0, false
		}
	}
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

	// Extract scheme from host if it includes http:// or https://
	if strings.HasPrefix(strings.ToLower(host), "http://") {
		scheme = "http"
		host = strings.TrimPrefix(host, "http://")
	} else if strings.HasPrefix(strings.ToLower(host), "https://") {
		scheme = "https"
		host = strings.TrimPrefix(host, "https://")
	}

	// Get grpcHost from config
	grpcHost, ok := cfg["grpcHost"].(string)
	if !ok {
		// If not provided, check if it's a Weaviate Cloud instance
		if strings.Contains(host, "weaviate.cloud") {
			// For Weaviate Cloud, prepend "grpc-" to the host
			grpcHost = "grpc-" + host
			// Ensure scheme is https for Weaviate Cloud
			scheme = "https"
		} else {
			return nil, fmt.Errorf("grpcHost is required in config")
		}
	}

	// Handle Weaviate Cloud instances
	if strings.Contains(host, "weaviate.cloud") && !strings.Contains(host, ":") {
		// Append port 443 if not specified for Weaviate Cloud
		host = host + ":443"
		// If grpcHost doesn't have a port, add it
		if !strings.Contains(grpcHost, ":") {
			grpcHost = grpcHost + ":443"
		}
		// Ensure scheme is https for Weaviate Cloud
		scheme = "https"
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
	if vectorConfig, ok := collectionConfig["vectorConfig"].(map[string]interface{}); ok {
		vectorConfigs := make(map[string]models.VectorConfig)
		for name, config := range vectorConfig {
			if configMap, ok := config.(map[string]interface{}); ok {
				vc := models.VectorConfig{}

				if vectorizer, ok := configMap["vectorizer"].(map[string]interface{}); ok {
					vc.Vectorizer = vectorizer
				}

				if vectorIndexType, ok := configMap["vectorIndexType"].(string); ok {
					vc.VectorIndexType = vectorIndexType
				}

				if vectorIndexConfig, ok := configMap["vectorIndexConfig"].(map[string]interface{}); ok {
					vc.VectorIndexConfig = vectorIndexConfig
				}

				vectorConfigs[name] = vc
			}
		}
		collection.VectorConfig = vectorConfigs
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

	// Updated multi-tenancy config
	if multiTenancy, ok := collectionConfig["multiTenancy"].(map[string]interface{}); ok {
		collection.MultiTenancyConfig = &models.MultiTenancyConfig{
			Enabled:              GetBoolValue(multiTenancy, "enabled", false),
			AutoTenantCreation:   GetBoolValue(multiTenancy, "autoTenantCreation", false),
			AutoTenantActivation: GetBoolValue(multiTenancy, "autoTenantActivation", false),
		}
	}

	// New replication config handling
	if replicationConfig, ok := collectionConfig["replicationConfig"].(map[string]interface{}); ok {
		// Handle factor type conversion safely
		var factor int64
		switch v := replicationConfig["factor"].(type) {
		case int64:
			factor = v
		case float64:
			factor = int64(v)
		case int:
			factor = int64(v)
		default:
			factor = 1 // Default value if type is unexpected
		}

		collection.ReplicationConfig = &models.ReplicationConfig{
			Factor:           factor,
			AsyncEnabled:     GetBoolValue(replicationConfig, "asyncEnabled", false),
			DeletionStrategy: GetStringValue(replicationConfig, "deletionStrategy"),
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

func (c *Client) DeleteAllCollections() error {
	return c.client.Schema().AllDeleter().Do(context.Background())
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

		batchDeleter = batchDeleter.WithWhere(where)
	}

	// Handle dry run option
	if dryRun, ok := options["dryRun"].(bool); ok {
		batchDeleter = batchDeleter.WithDryRun(dryRun)
	}

	// Handle output format
	if output, ok := options["output"].(string); ok {
		batchDeleter = batchDeleter.WithOutput(output)
	}

	// Handle tenant
	if tenant, ok := options["tenant"].(string); ok {
		batchDeleter = batchDeleter.WithTenant(tenant)
	}

	replicationMap := map[string]string{
		"all":    replication.ConsistencyLevel.ALL,
		"one":    replication.ConsistencyLevel.ONE,
		"quorum": replication.ConsistencyLevel.QUORUM,
	}

	// Handle consistency level
	if consistencyLevel, ok := options["consistencyLevel"].(string); ok {
		batchDeleter = batchDeleter.WithConsistencyLevel(replicationMap[consistencyLevel])
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

func (c *Client) ObjectInsert(className string, object map[string]interface{}) (map[string]interface{}, error) {
	creator := c.client.Data().Creator().WithClassName(className)

	// Optional ID
	if id, ok := object["id"].(string); ok {
		creator = creator.WithID(id)
	}

	// Properties handling
	if props, ok := object["properties"].(map[string]interface{}); ok {
		creator = creator.WithProperties(props)
	}

	// Vector handling (single vector)
	if vector, ok := object["vector"].([]interface{}); ok {
		float32Vec := make([]float32, len(vector))
		for i, v := range vector {
			if f, ok := v.(float64); ok {
				float32Vec[i] = float32(f)
			}
		}
		creator = creator.WithVector(float32Vec)
	}

	// Named vectors handling
	if vectors, ok := object["vectors"].(map[string]interface{}); ok {
		namedVectors := make(models.Vectors)
		for name, vec := range vectors {
			if vecSlice, ok := vec.([]interface{}); ok {
				float32Vec := make([]float32, len(vecSlice))
				for i, v := range vecSlice {
					if f, ok := v.(float64); ok {
						float32Vec[i] = float32(f)
					}
				}
				namedVectors[name] = float32Vec
			}
		}
		creator = creator.WithVectors(namedVectors)
	}

	// Tenant handling
	if tenant, ok := object["tenant"].(string); ok {
		creator = creator.WithTenant(tenant)
	}

	// Consistency level handling
	replicationMap := map[string]string{
		"all":    replication.ConsistencyLevel.ALL,
		"one":    replication.ConsistencyLevel.ONE,
		"quorum": replication.ConsistencyLevel.QUORUM,
	}
	// if consistencyLevel does not match, throw an error
	if cl, ok := object["consistencyLevel"].(string); ok {
		if _, ok := replicationMap[cl]; !ok {
			return nil, fmt.Errorf("invalid consistency level: %s", cl)
		}
		creator = creator.WithConsistencyLevel(replicationMap[cl])
	}

	// Execute the insert
	wrapper, err := creator.Do(context.Background())
	if err != nil {
		return nil, err
	}

	// Build result map
	result := map[string]interface{}{
		"id":         wrapper.Object.ID.String(),
		"properties": wrapper.Object.Properties,
	}

	// Include vector/vectors if present
	if len(wrapper.Object.Vector) > 0 {
		result["vector"] = wrapper.Object.Vector
	}
	if len(wrapper.Object.Vectors) > 0 {
		result["vectors"] = wrapper.Object.Vectors
	}

	// Add tenant if specified
	if wrapper.Object.Tenant != "" {
		result["tenant"] = wrapper.Object.Tenant
	}

	return result, nil
}

func (c *Client) FetchObjects(className string, options map[string]interface{}) (map[string]interface{}, error) {
	getter := c.client.Data().ObjectsGetter().WithClassName(className)

	// Handle ID if provided
	if id, ok := options["id"].(string); ok {
		getter = getter.WithID(id)
	}

	// Universal number conversion for limit
	if limitVal, exists := options["limit"]; exists {
		if limit, ok := ToInt(limitVal); ok {
			getter = getter.WithLimit(limit)
		}
	}

	// Universal number conversion for offset
	if offsetVal, exists := options["offset"]; exists {
		if offset, ok := ToInt(offsetVal); ok {
			getter = getter.WithOffset(offset)
		}
	}

	// Handle cursor pagination
	if after, ok := options["after"].(string); ok {
		getter = getter.WithAfter(after)
	}

	// Handle consistency level
	if cl, ok := options["consistencyLevel"].(string); ok {
		getter = getter.WithConsistencyLevel(cl)
	}

	// Handle tenant
	if tenant, ok := options["tenant"].(string); ok {
		getter = getter.WithTenant(tenant)
	}

	// Handle node name
	if nodeName, ok := options["nodeName"].(string); ok {
		getter = getter.WithNodeName(nodeName)
	}

	// Handle additional properties
	if additional, ok := options["additional"].([]interface{}); ok {
		// Convert []interface{} to []string
		additionalProps := make([]string, len(additional))
		for i, prop := range additional {
			if strProp, ok := prop.(string); ok {
				additionalProps[i] = strProp
			}
		}

		// Now process the string slice
		for _, prop := range additionalProps {
			// Handle special cases according to documentation
			if prop == "vector" {
				getter = getter.WithVector()
			} else if prop == "id" {
				// Skip "id" as it's returned by default and not a valid additional property
				continue
			} else {
				getter = getter.WithAdditional(prop)
			}
		}
	} else if additional, ok := options["additional"].([]string); ok {
		// Handle direct []string type for compatibility
		for _, prop := range additional {
			// Handle special cases according to documentation
			if prop == "vector" {
				getter = getter.WithVector()
			} else if prop == "id" {
				// Skip "id" as it's returned by default and not a valid additional property
				continue
			} else {
				getter = getter.WithAdditional(prop)
			}
		}
	}

	// Execute the query
	objects, err := getter.Do(context.Background())
	if err != nil {
		return nil, err
	}

	// Convert results to simplified map for JS
	result := make(map[string]interface{})
	objectsList := make([]map[string]interface{}, len(objects))

	for i, obj := range objects {
		item := map[string]interface{}{
			"id":         obj.ID.String(),
			"properties": obj.Properties,
		}

		if len(obj.Vector) > 0 {
			item["vector"] = obj.Vector
		}
		if len(obj.Vectors) > 0 {
			// Convert models.Vectors to a map that can be serialized to JSON
			vectorsMap := make(map[string]interface{})
			for name, vec := range obj.Vectors {
				vectorsMap[name] = vec
			}
			item["vectors"] = vectorsMap
		}
		if obj.Additional != nil {
			item["additional"] = obj.Additional
		}

		objectsList[i] = item
	}

	result["objects"] = objectsList
	return result, nil
}
