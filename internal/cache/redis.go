package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"time"

	"github.com/yejune/go-react-ssr/internal/reactbuilder"
	"github.com/redis/go-redis/v9"
)

// RedisCache provides distributed caching via Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// RedisConfig configures the Redis cache
type RedisConfig struct {
	Addr     string        // Redis address (e.g., "localhost:6379")
	Password string        // Redis password (empty for no auth)
	DB       int           // Redis database number
	TTL      time.Duration // Cache TTL (0 = no expiration)
	Prefix   string        // Key prefix (default: "gossr:")
	UseTLS   bool          // Enable TLS connection
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(config RedisConfig) (*RedisCache, error) {
	opts := &redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	}

	// Enable TLS if configured
	if config.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	prefix := config.Prefix
	if prefix == "" {
		prefix = "gossr:"
	}

	return &RedisCache{
		client: client,
		ttl:    config.TTL,
		prefix: prefix,
	}, nil
}

// GetServerBuild retrieves a server build from Redis
func (rc *RedisCache) GetServerBuild(filePath string) (reactbuilder.BuildResult, bool) {
	ctx := context.Background()
	key := rc.prefix + "server:" + filePath
	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		return reactbuilder.BuildResult{}, false
	}

	var result reactbuilder.BuildResult
	if err := json.Unmarshal(data, &result); err != nil {
		return reactbuilder.BuildResult{}, false
	}

	return result, true
}

// SetServerBuild stores a server build in Redis
func (rc *RedisCache) SetServerBuild(filePath string, build reactbuilder.BuildResult) {
	ctx := context.Background()
	key := rc.prefix + "server:" + filePath
	data, err := json.Marshal(build)
	if err != nil {
		return
	}

	rc.client.Set(ctx, key, data, rc.ttl)
}

// RemoveServerBuild removes a server build from Redis
func (rc *RedisCache) RemoveServerBuild(filePath string) {
	ctx := context.Background()
	key := rc.prefix + "server:" + filePath
	rc.client.Del(ctx, key)
}

// GetClientBuild retrieves a client build from Redis
func (rc *RedisCache) GetClientBuild(filePath string) (reactbuilder.BuildResult, bool) {
	ctx := context.Background()
	key := rc.prefix + "client:" + filePath
	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		return reactbuilder.BuildResult{}, false
	}

	var result reactbuilder.BuildResult
	if err := json.Unmarshal(data, &result); err != nil {
		return reactbuilder.BuildResult{}, false
	}

	return result, true
}

// SetClientBuild stores a client build in Redis
func (rc *RedisCache) SetClientBuild(filePath string, build reactbuilder.BuildResult) {
	ctx := context.Background()
	key := rc.prefix + "client:" + filePath
	data, err := json.Marshal(build)
	if err != nil {
		return
	}

	rc.client.Set(ctx, key, data, rc.ttl)
}

// RemoveClientBuild removes a client build from Redis
func (rc *RedisCache) RemoveClientBuild(filePath string) {
	ctx := context.Background()
	key := rc.prefix + "client:" + filePath
	rc.client.Del(ctx, key)
}

// SetParentFile maps a routeID to a parent file path
func (rc *RedisCache) SetParentFile(routeID, filePath string) {
	ctx := context.Background()
	key := rc.prefix + "routes"
	rc.client.HSet(ctx, key, routeID, filePath)
}

// GetRouteIDSForParentFile returns all route IDs for a given file path
func (rc *RedisCache) GetRouteIDSForParentFile(filePath string) []string {
	ctx := context.Background()
	key := rc.prefix + "routes"
	result, err := rc.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil
	}

	var routes []string
	for route, file := range result {
		if file == filePath {
			routes = append(routes, route)
		}
	}
	return routes
}

// GetAllRouteIDS returns all route IDs
func (rc *RedisCache) GetAllRouteIDS() []string {
	ctx := context.Background()
	key := rc.prefix + "routes"
	result, err := rc.client.HKeys(ctx, key).Result()
	if err != nil {
		return nil
	}
	return result
}

// GetRouteIDSWithFile returns route IDs associated with a file
func (rc *RedisCache) GetRouteIDSWithFile(filePath string) []string {
	reactFilesWithDependency := rc.GetParentFilesFromDependency(filePath)
	if len(reactFilesWithDependency) == 0 {
		reactFilesWithDependency = []string{filePath}
	}
	var routeIDS []string
	for _, reactFile := range reactFilesWithDependency {
		routeIDS = append(routeIDS, rc.GetRouteIDSForParentFile(reactFile)...)
	}
	return routeIDS
}

// SetParentFileDependencies sets dependencies for a parent file
func (rc *RedisCache) SetParentFileDependencies(filePath string, dependencies []string) {
	ctx := context.Background()
	key := rc.prefix + "deps:" + filePath
	data, _ := json.Marshal(dependencies)
	rc.client.Set(ctx, key, data, rc.ttl)
}

// GetParentFilesFromDependency returns parent files that depend on a given file
func (rc *RedisCache) GetParentFilesFromDependency(dependencyPath string) []string {
	ctx := context.Background()
	pattern := rc.prefix + "deps:*"
	var parentFilePaths []string

	var cursor uint64
	for {
		keys, nextCursor, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}

		for _, key := range keys {
			data, err := rc.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			var deps []string
			if err := json.Unmarshal(data, &deps); err != nil {
				continue
			}

			for _, dep := range deps {
				if dep == dependencyPath {
					// Extract parent file path from key
					parentPath := key[len(rc.prefix+"deps:"):]
					parentFilePaths = append(parentFilePaths, parentPath)
					break
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return parentFilePaths
}

// Clear removes all gossr keys from cache
func (rc *RedisCache) Clear() {
	ctx := context.Background()
	pattern := rc.prefix + "*"
	var cursor uint64
	for {
		keys, nextCursor, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}

		if len(keys) > 0 {
			rc.client.Del(ctx, keys...)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

// Invalidate removes a specific key from cache
func (rc *RedisCache) Invalidate(filePath string) {
	ctx := context.Background()
	keys := []string{
		rc.prefix + "server:" + filePath,
		rc.prefix + "client:" + filePath,
	}
	rc.client.Del(ctx, keys...)
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Stats returns cache statistics
func (rc *RedisCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := rc.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	// Count gossr keys
	pattern := rc.prefix + "*"
	var count int64
	var cursor uint64
	for {
		keys, nextCursor, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return map[string]interface{}{
		"type":       "redis",
		"key_count":  count,
		"prefix":     rc.prefix,
		"redis_info": info,
	}, nil
}
