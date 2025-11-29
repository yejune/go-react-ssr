package cache

import (
	"sync"

	"github.com/yejune/go-react-ssr/internal/reactbuilder"
)

// LocalCache is an in-memory cache implementation
// It implements the Cache interface
type LocalCache struct {
	serverBuilds             *serverBuilds
	clientBuilds             *clientBuilds
	routeIDToParentFile      *routeIDToParentFile
	parentFileToDependencies *parentFileToDependencies
}

// NewLocalCache creates a new in-memory cache
func NewLocalCache() *LocalCache {
	return &LocalCache{
		serverBuilds: &serverBuilds{
			builds: make(map[string]reactbuilder.BuildResult),
			lock:   sync.RWMutex{},
		},
		clientBuilds: &clientBuilds{
			builds: make(map[string]reactbuilder.BuildResult),
			lock:   sync.RWMutex{},
		},
		routeIDToParentFile: &routeIDToParentFile{
			reactFiles: make(map[string]string),
			lock:       sync.RWMutex{},
		},
		parentFileToDependencies: &parentFileToDependencies{
			dependencies: make(map[string][]string),
			lock:         sync.RWMutex{},
		},
	}
}

type serverBuilds struct {
	builds map[string]reactbuilder.BuildResult
	lock   sync.RWMutex
}

func (cm *LocalCache) GetServerBuild(filePath string) (reactbuilder.BuildResult, bool) {
	cm.serverBuilds.lock.RLock()
	defer cm.serverBuilds.lock.RUnlock()
	build, ok := cm.serverBuilds.builds[filePath]
	return build, ok
}

func (cm *LocalCache) SetServerBuild(filePath string, build reactbuilder.BuildResult) {
	cm.serverBuilds.lock.Lock()
	defer cm.serverBuilds.lock.Unlock()
	cm.serverBuilds.builds[filePath] = build
}

func (cm *LocalCache) RemoveServerBuild(filePath string) {
	cm.serverBuilds.lock.Lock()
	defer cm.serverBuilds.lock.Unlock()
	if _, ok := cm.serverBuilds.builds[filePath]; !ok {
		return
	}
	delete(cm.serverBuilds.builds, filePath)
}

type clientBuilds struct {
	builds map[string]reactbuilder.BuildResult
	lock   sync.RWMutex
}

func (cm *LocalCache) GetClientBuild(filePath string) (reactbuilder.BuildResult, bool) {
	cm.clientBuilds.lock.RLock()
	defer cm.clientBuilds.lock.RUnlock()
	build, ok := cm.clientBuilds.builds[filePath]
	return build, ok
}

func (cm *LocalCache) SetClientBuild(filePath string, build reactbuilder.BuildResult) {
	cm.clientBuilds.lock.Lock()
	defer cm.clientBuilds.lock.Unlock()
	cm.clientBuilds.builds[filePath] = build
}

func (cm *LocalCache) RemoveClientBuild(filePath string) {
	cm.clientBuilds.lock.Lock()
	defer cm.clientBuilds.lock.Unlock()
	if _, ok := cm.clientBuilds.builds[filePath]; !ok {
		return
	}
	delete(cm.clientBuilds.builds, filePath)
}

type routeIDToParentFile struct {
	reactFiles map[string]string
	lock       sync.RWMutex
}

func (cm *LocalCache) SetParentFile(routeID, filePath string) {
	cm.routeIDToParentFile.lock.Lock()
	defer cm.routeIDToParentFile.lock.Unlock()
	cm.routeIDToParentFile.reactFiles[routeID] = filePath
}

func (cm *LocalCache) GetRouteIDSForParentFile(filePath string) []string {
	cm.routeIDToParentFile.lock.RLock()
	defer cm.routeIDToParentFile.lock.RUnlock()
	var routes []string
	for route, file := range cm.routeIDToParentFile.reactFiles {
		if file == filePath {
			routes = append(routes, route)
		}
	}
	return routes
}

func (cm *LocalCache) GetAllRouteIDS() []string {
	cm.routeIDToParentFile.lock.RLock()
	defer cm.routeIDToParentFile.lock.RUnlock()
	routes := make([]string, 0, len(cm.routeIDToParentFile.reactFiles))
	for route := range cm.routeIDToParentFile.reactFiles {
		routes = append(routes, route)
	}
	return routes
}

func (cm *LocalCache) GetRouteIDSWithFile(filePath string) []string {
	reactFilesWithDependency := cm.GetParentFilesFromDependency(filePath)
	if len(reactFilesWithDependency) == 0 {
		reactFilesWithDependency = []string{filePath}
	}
	var routeIDS []string
	for _, reactFile := range reactFilesWithDependency {
		routeIDS = append(routeIDS, cm.GetRouteIDSForParentFile(reactFile)...)
	}
	return routeIDS
}

type parentFileToDependencies struct {
	dependencies map[string][]string
	lock         sync.RWMutex
}

func (cm *LocalCache) SetParentFileDependencies(filePath string, dependencies []string) {
	cm.parentFileToDependencies.lock.Lock()
	defer cm.parentFileToDependencies.lock.Unlock()
	cm.parentFileToDependencies.dependencies[filePath] = dependencies
}

func (cm *LocalCache) GetParentFilesFromDependency(dependencyPath string) []string {
	cm.parentFileToDependencies.lock.RLock()
	defer cm.parentFileToDependencies.lock.RUnlock()
	var parentFilePaths []string
	for parentFilePath, dependencies := range cm.parentFileToDependencies.dependencies {
		for _, dependency := range dependencies {
			if dependency == dependencyPath {
				parentFilePaths = append(parentFilePaths, parentFilePath)
			}
		}
	}
	return parentFilePaths
}

// Clear removes all cached data
func (cm *LocalCache) Clear() {
	cm.serverBuilds.lock.Lock()
	cm.serverBuilds.builds = make(map[string]reactbuilder.BuildResult)
	cm.serverBuilds.lock.Unlock()

	cm.clientBuilds.lock.Lock()
	cm.clientBuilds.builds = make(map[string]reactbuilder.BuildResult)
	cm.clientBuilds.lock.Unlock()

	cm.routeIDToParentFile.lock.Lock()
	cm.routeIDToParentFile.reactFiles = make(map[string]string)
	cm.routeIDToParentFile.lock.Unlock()

	cm.parentFileToDependencies.lock.Lock()
	cm.parentFileToDependencies.dependencies = make(map[string][]string)
	cm.parentFileToDependencies.lock.Unlock()
}

// Manager is an alias for LocalCache for backward compatibility
type Manager = LocalCache

// NewManager creates a new LocalCache (for backward compatibility)
func NewManager() *LocalCache {
	return NewLocalCache()
}

// NewCache creates a cache based on the config
func NewCache(config CacheConfig) (Cache, error) {
	switch config.Type {
	case CacheTypeRedis:
		return NewRedisCache(RedisConfig{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
			UseTLS:   config.RedisTLS,
		})
	case CacheTypeLocal, "":
		return NewLocalCache(), nil
	default:
		return NewLocalCache(), nil
	}
}
