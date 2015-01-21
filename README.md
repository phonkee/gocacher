# Gocacher

Gocacher is cache abstraction. It's intended to use in web applications with
possibility to choose cache implementation directly from configuration (dsn).

Currently redis implementation is written and tested, but in the future there will
be more implementations (memcached, and other..).

All cache implementations satisfy this interface
```go
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

## Contribute
Welcome!