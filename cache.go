package gocacher

import (
	"fmt"
	"sort"
	"time"

	"github.com/phonkee/godsn"
)

/*
CacheDriver interface
 	All cache drivers must satisfy this interface.
*/
type CacheDriver interface {
	Open(dsn string) (Cache, error)

	OpenConnection(connection interface{}, settings ...string) (Cache, error)
}

/*
Cache interface
 	All cache implementations must satisfy this interface.
*/
type Cache interface {
	// returns cache value by key
	Get(key string) ([]byte, error)

	// Sets key to value, if expiration is not given it's used from settings
	Set(key string, value []byte, expiration ...time.Duration) error

	// Deletes key in cache
	Delete(key string) error

	// Increments key by 1, if num is given by that amout will be incremented
	Incr(key string, num ...int) (int, error)

	// Decrements key by 1, if num is given it decrements by given number
	Decr(key string, num ...int) (int, error)

	// Return cache to cache pool
	Close() error
}

// registered cache drivers
var drivers = make(map[string]CacheDriver)

/*
Register makes a cache driver available by the provided name.
If Register is called twice with the same name or if driver is nil,
it panics.
*/
func Register(name string, driver CacheDriver) {
	if driver == nil {
		panic("gocacher: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("gocacher: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// returns list of available driver names
func Drivers() []string {
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

/*
Opens cache by dsn
This method is preferred to use for instantiate Cache.
DSN is used as connector. Additional settings are set as url parameters.

Example:
	redis://localhost:5379/0?pool_max_active=10&expiration=10

Which translates to connection to localhost:5379 redis pool max_active=10
and expiration for cache.Set default value will be set to 10 seconds

*/
func Open(dsn string) (Cache, error) {
	d, err := godsn.Parse(dsn)
	if err != nil {
		return nil, err
	}
	di, ok := drivers[d.Scheme()]
	if !ok {
		return nil, fmt.Errorf("gocacher: unknown driver %q (forgotten import?)", d.Scheme())
	}

	return di.Open(dsn)
}

// opens message queue by name and connection
func OpenConnection(driver string, connection interface{}, settings ...string) (Cache, error) {
	di, ok := drivers[driver]
	if !ok {
		return nil, fmt.Errorf("gocacher: unknown driver %q (forgotten import?)", driver)
	}
	// additional settings
	urlSettings := ""
	if len(settings) > 0 {
		urlSettings = settings[0]
	}

	return di.OpenConnection(connection, urlSettings)
}
