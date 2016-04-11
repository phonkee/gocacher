# Gocacher

[![Build Status](https://travis-ci.org/phonkee/gocacher.svg?branch=master)](https://travis-ci.org/phonkee/gocacher)


Gocacher is cache abstraction. It's intended to use in web applications with
possibility to choose cache implementation directly from configuration (dsn).

Currently redis implementation is written and tested, but in the future there will
be more implementations (memcached, and other..).

All cache implementations satisfy this interface
```go
type Cacher interface {
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
```


Before we dive into examples all use this import
```go
import (
	"time"
	"github.com/phonkee/gocacher"
)
```

Examples:

```go
// Open Cache with expiration of 60 seconds and prefix "cache"
cache, _ := gocacher.Open("redis://localhost:5379/1?expiration=60&prefix=cache")

// this key will be set with expiration set in url "expiration=60"
cache.Set("key", []byte("message"))

// value will be []byte("message")
value, _ := cache.Get("key")

// this key will expire in 5 seconds
cache.Set("key-with-expiration", []byte("message"), time.Seconds*5)

// deletes the key from cache
cache.Delete("key")

// increments key and returns incremented value
i, _ := cache.Incr("increment-key") // i is now 1
i, _ := cache.Incr("increment-key", 10) // i is now 11

i, _ := cache.Decr("increment-key", 5) // i is now 6
i, _ := cache.Decr("increment-key") // i is now 5

```

### Open Cache connection

You can open connection in two ways.

```go
cache, _ := gocacher.Open("redis://localhost:5379/1")
```

or you can provide connection to gocacher.OpenConnection and provide additional settings
as url query

```go
pool := &redis.Pool{
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", "localhost:5379")
	},
}

cache, _ := gocacher.OpenConnection("redis", pool)

// With additional cache settings
cache, _ := gocacher.OpenConnection("redis", pool, "expiration=60&prefix=cache")
```

#### Parameters
You can pass multiple parameters to `Open` or `OpenConnection` as query part of dsn.

e.g.
```go
cache, _ := gocacher.Open("locmem:///0?expiration=10s")
cache, _ := gocacher.Open("redis://localhost:5379/1?expiration=60&prefix=cache")
```

### Cache implementations

Currently two imlpementations are available:
* redis - redis storage
* locmem - local memory storage


#### Locmem
Local memory cache. Supports multiple databases. Currently there is no garbage collect or limiting of items in database.
This will be updated in near future.

```go
cache, _ := gocacher.Open("locmem:///0")
```

##### parameters:
* expiration - string parsed with time parse duration. (default expiration is 0s - neverending)

#### Redis
Redis cache support. Supports multiple databases and all commands.

##### parameters:
* pool_max_active - redis pool settings
* pool_max_idle - redis pool settings
* pool_idle_timeout - redis pool settings
* expiration - default expiration
* prefix - prefix for cache keys

## Contribute
Welcome!