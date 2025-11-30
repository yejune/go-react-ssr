package cache

import (
	"container/list"
	"sync"

	"github.com/yejune/gotossr/internal/reactbuilder"
)

// MaxCacheEntries is the maximum number of builds to cache per type
const MaxCacheEntries = 100

// LocalCache is an in-memory cache implementation
// It implements the Cache interface
type LocalCache struct {
	serverBuilds             *serverBuilds
	clientBuilds             *clientBuilds
	routeIDToParentFile      *routeIDToParentFile
	parentFileToDependencies *parentFileToDependencies
	// Reverse index: dependency -> parent files
	dependencyToParentFiles *dependencyToParentFiles
}

// NewLocalCache creates a new in-memory cache
func NewLocalCache() *LocalCache {
	return &LocalCache{
		serverBuilds: &serverBuilds{
			builds:   make(map[string]reactbuilder.BuildResult),
			lruList:  list.New(),
			lruIndex: make(map[string]*list.Element),
			lock:     sync.RWMutex{},
		},
		clientBuilds: &clientBuilds{
			builds:   make(map[string]reactbuilder.BuildResult),
			lruList:  list.New(),
			lruIndex: make(map[string]*list.Element),
			lock:     sync.RWMutex{},
		},
		routeIDToParentFile: &routeIDToParentFile{
			reactFiles: make(map[string]string),
			lock:       sync.RWMutex{},
		},
		parentFileToDependencies: &parentFileToDependencies{
			dependencies: make(map[string][]string),
			lock:         sync.RWMutex{},
		},
		dependencyToParentFiles: &dependencyToParentFiles{
			parents: make(map[string]map[string]struct{}),
			lock:    sync.RWMutex{},
		},
	}
}

type serverBuilds struct {
	builds   map[string]reactbuilder.BuildResult
	lruList  *list.List                    // LRU order (front = most recent)
	lruIndex map[string]*list.Element      // filePath -> list element
	lock     sync.RWMutex
}

func (cm *LocalCache) GetServerBuild(filePath string) (reactbuilder.BuildResult, bool, error) {
	cm.serverBuilds.lock.Lock()
	defer cm.serverBuilds.lock.Unlock()
	build, ok := cm.serverBuilds.builds[filePath]
	if ok {
		// Move to front (most recently used)
		if elem, exists := cm.serverBuilds.lruIndex[filePath]; exists {
			cm.serverBuilds.lruList.MoveToFront(elem)
		}
	}
	return build, ok, nil
}

func (cm *LocalCache) SetServerBuild(filePath string, build reactbuilder.BuildResult) error {
	cm.serverBuilds.lock.Lock()
	defer cm.serverBuilds.lock.Unlock()

	// If already exists, update and move to front
	if elem, exists := cm.serverBuilds.lruIndex[filePath]; exists {
		cm.serverBuilds.lruList.MoveToFront(elem)
		cm.serverBuilds.builds[filePath] = build
		return nil
	}

	// Evict if at capacity
	if len(cm.serverBuilds.builds) >= MaxCacheEntries {
		oldest := cm.serverBuilds.lruList.Back()
		if oldest != nil {
			oldPath := oldest.Value.(string)
			delete(cm.serverBuilds.builds, oldPath)
			delete(cm.serverBuilds.lruIndex, oldPath)
			cm.serverBuilds.lruList.Remove(oldest)
		}
	}

	// Add new entry
	cm.serverBuilds.builds[filePath] = build
	elem := cm.serverBuilds.lruList.PushFront(filePath)
	cm.serverBuilds.lruIndex[filePath] = elem
	return nil
}

func (cm *LocalCache) RemoveServerBuild(filePath string) error {
	cm.serverBuilds.lock.Lock()
	defer cm.serverBuilds.lock.Unlock()
	delete(cm.serverBuilds.builds, filePath)
	if elem, exists := cm.serverBuilds.lruIndex[filePath]; exists {
		cm.serverBuilds.lruList.Remove(elem)
		delete(cm.serverBuilds.lruIndex, filePath)
	}
	return nil
}

type clientBuilds struct {
	builds   map[string]reactbuilder.BuildResult
	lruList  *list.List
	lruIndex map[string]*list.Element
	lock     sync.RWMutex
}

func (cm *LocalCache) GetClientBuild(filePath string) (reactbuilder.BuildResult, bool, error) {
	cm.clientBuilds.lock.Lock()
	defer cm.clientBuilds.lock.Unlock()
	build, ok := cm.clientBuilds.builds[filePath]
	if ok {
		if elem, exists := cm.clientBuilds.lruIndex[filePath]; exists {
			cm.clientBuilds.lruList.MoveToFront(elem)
		}
	}
	return build, ok, nil
}

func (cm *LocalCache) SetClientBuild(filePath string, build reactbuilder.BuildResult) error {
	cm.clientBuilds.lock.Lock()
	defer cm.clientBuilds.lock.Unlock()

	if elem, exists := cm.clientBuilds.lruIndex[filePath]; exists {
		cm.clientBuilds.lruList.MoveToFront(elem)
		cm.clientBuilds.builds[filePath] = build
		return nil
	}

	if len(cm.clientBuilds.builds) >= MaxCacheEntries {
		oldest := cm.clientBuilds.lruList.Back()
		if oldest != nil {
			oldPath := oldest.Value.(string)
			delete(cm.clientBuilds.builds, oldPath)
			delete(cm.clientBuilds.lruIndex, oldPath)
			cm.clientBuilds.lruList.Remove(oldest)
		}
	}

	cm.clientBuilds.builds[filePath] = build
	elem := cm.clientBuilds.lruList.PushFront(filePath)
	cm.clientBuilds.lruIndex[filePath] = elem
	return nil
}

func (cm *LocalCache) RemoveClientBuild(filePath string) error {
	cm.clientBuilds.lock.Lock()
	defer cm.clientBuilds.lock.Unlock()
	delete(cm.clientBuilds.builds, filePath)
	if elem, exists := cm.clientBuilds.lruIndex[filePath]; exists {
		cm.clientBuilds.lruList.Remove(elem)
		delete(cm.clientBuilds.lruIndex, filePath)
	}
	return nil
}

type routeIDToParentFile struct {
	reactFiles map[string]string
	lock       sync.RWMutex
}

func (cm *LocalCache) SetParentFile(routeID, filePath string) error {
	cm.routeIDToParentFile.lock.Lock()
	defer cm.routeIDToParentFile.lock.Unlock()
	cm.routeIDToParentFile.reactFiles[routeID] = filePath
	return nil
}

func (cm *LocalCache) GetRouteIDSForParentFile(filePath string) ([]string, error) {
	cm.routeIDToParentFile.lock.RLock()
	defer cm.routeIDToParentFile.lock.RUnlock()
	var routes []string
	for route, file := range cm.routeIDToParentFile.reactFiles {
		if file == filePath {
			routes = append(routes, route)
		}
	}
	return routes, nil
}

func (cm *LocalCache) GetAllRouteIDS() ([]string, error) {
	cm.routeIDToParentFile.lock.RLock()
	defer cm.routeIDToParentFile.lock.RUnlock()
	routes := make([]string, 0, len(cm.routeIDToParentFile.reactFiles))
	for route := range cm.routeIDToParentFile.reactFiles {
		routes = append(routes, route)
	}
	return routes, nil
}

func (cm *LocalCache) GetRouteIDSWithFile(filePath string) ([]string, error) {
	reactFilesWithDependency, err := cm.GetParentFilesFromDependency(filePath)
	if err != nil {
		return nil, err
	}
	if len(reactFilesWithDependency) == 0 {
		reactFilesWithDependency = []string{filePath}
	}
	var routeIDS []string
	for _, reactFile := range reactFilesWithDependency {
		routes, err := cm.GetRouteIDSForParentFile(reactFile)
		if err != nil {
			return nil, err
		}
		routeIDS = append(routeIDS, routes...)
	}
	return routeIDS, nil
}

type parentFileToDependencies struct {
	dependencies map[string][]string
	lock         sync.RWMutex
}

// dependencyToParentFiles is a reverse index for O(1) lookup
type dependencyToParentFiles struct {
	parents map[string]map[string]struct{} // dependency -> set of parent files
	lock    sync.RWMutex
}

func (cm *LocalCache) SetParentFileDependencies(filePath string, dependencies []string) error {
	// Update forward index
	cm.parentFileToDependencies.lock.Lock()
	oldDeps := cm.parentFileToDependencies.dependencies[filePath]
	cm.parentFileToDependencies.dependencies[filePath] = dependencies
	cm.parentFileToDependencies.lock.Unlock()

	// Update reverse index
	cm.dependencyToParentFiles.lock.Lock()
	defer cm.dependencyToParentFiles.lock.Unlock()

	// Remove old reverse mappings
	for _, dep := range oldDeps {
		if parents, ok := cm.dependencyToParentFiles.parents[dep]; ok {
			delete(parents, filePath)
			if len(parents) == 0 {
				delete(cm.dependencyToParentFiles.parents, dep)
			}
		}
	}

	// Add new reverse mappings
	for _, dep := range dependencies {
		if cm.dependencyToParentFiles.parents[dep] == nil {
			cm.dependencyToParentFiles.parents[dep] = make(map[string]struct{})
		}
		cm.dependencyToParentFiles.parents[dep][filePath] = struct{}{}
	}

	return nil
}

func (cm *LocalCache) GetParentFilesFromDependency(dependencyPath string) ([]string, error) {
	cm.dependencyToParentFiles.lock.RLock()
	defer cm.dependencyToParentFiles.lock.RUnlock()

	parents, ok := cm.dependencyToParentFiles.parents[dependencyPath]
	if !ok {
		return nil, nil
	}

	result := make([]string, 0, len(parents))
	for parent := range parents {
		result = append(result, parent)
	}
	return result, nil
}

// Clear removes all cached data
func (cm *LocalCache) Clear() error {
	cm.serverBuilds.lock.Lock()
	cm.serverBuilds.builds = make(map[string]reactbuilder.BuildResult)
	cm.serverBuilds.lruList = list.New()
	cm.serverBuilds.lruIndex = make(map[string]*list.Element)
	cm.serverBuilds.lock.Unlock()

	cm.clientBuilds.lock.Lock()
	cm.clientBuilds.builds = make(map[string]reactbuilder.BuildResult)
	cm.clientBuilds.lruList = list.New()
	cm.clientBuilds.lruIndex = make(map[string]*list.Element)
	cm.clientBuilds.lock.Unlock()

	cm.routeIDToParentFile.lock.Lock()
	cm.routeIDToParentFile.reactFiles = make(map[string]string)
	cm.routeIDToParentFile.lock.Unlock()

	cm.parentFileToDependencies.lock.Lock()
	cm.parentFileToDependencies.dependencies = make(map[string][]string)
	cm.parentFileToDependencies.lock.Unlock()

	cm.dependencyToParentFiles.lock.Lock()
	cm.dependencyToParentFiles.parents = make(map[string]map[string]struct{})
	cm.dependencyToParentFiles.lock.Unlock()

	return nil
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
