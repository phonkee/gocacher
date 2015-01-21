package gocacher

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/phonkee/godsn"
)

const (
	URL_PARAM_POOL_MAX_ACTIVE   = "pool_max_active"
	URL_PARAM_POOL_MAX_IDLE     = "pool_max_idle"
	URL_PARAM_POOL_IDLE_TIMEOUT = "pool_idle_timeout"
	URL_PARAM_EXPIRATION        = "expiration"
	URL_PARAM_PREFIX            = "prefix"

	DEFAULT_POOL_MAX_ACTIVE   = 20
	DEFAULT_POOL_MAX_IDLE     = 10
	DEFAULT_POOL_IDLE_TIMEOUT = 200 * time.Millisecond
	DEFAULT_EXPIRATION        = 0 * time.Second
	DEFAULT_PREFIX            = ""
)

func init() {
	Register("redis", &RedisDriver{})
}

type RedisDriver struct{}

func (r *RedisDriver) Open(dsn string) (Cache, error) {
	d, err := ParseRedisDSN(dsn)
	if err != nil {
		return nil, err
	}
	cache := RedisCache{
		pool:       d.Pool(),
		expiration: d.settings.expiration,
		settings:   d.settings,
	}

	return &cache, nil
}

func (r *RedisDriver) OpenConnection(connection interface{}, settings ...string) (Cache, error) {

	switch connection.(type) {
	case *redis.Pool:
		break
	default:
		return nil, fmt.Errorf("Connection %s is unknown.", connection)
	}

	s, err := NewRedisSettingsFromQuery(settings[0])
	if err != nil {
		return nil, err
	}

	cache := RedisCache{
		pool:       connection.(*redis.Pool),
		expiration: s.expiration,
		settings:   s,
	}

	return &cache, nil
}

type PoolSettings struct {
	MaxIdle     int
	MaxActive   int
	IdleTimeout time.Duration
}

type RedisSettings struct {
	pool       PoolSettings
	expiration time.Duration
	prefix     string
}

// returns prefixed string
func (r *RedisSettings) Prefixed(str string) string {
	if r.prefix == "" {
		return str
	}
	return r.prefix + ":" + str
}

func NewRedisSettings(values *godsn.DSNValues) (*RedisSettings, error) {
	settings := RedisSettings{
		pool: PoolSettings{
			MaxIdle:     10,
			MaxActive:   20,
			IdleTimeout: 200,
		},
		expiration: DEFAULT_EXPIRATION,
		prefix:     DEFAULT_PREFIX,
	}

	settings.pool.MaxActive = values.GetInt(
		URL_PARAM_POOL_MAX_ACTIVE,
		DEFAULT_POOL_MAX_ACTIVE)

	settings.pool.MaxIdle = values.GetInt(
		URL_PARAM_POOL_MAX_IDLE,
		DEFAULT_POOL_MAX_IDLE)

	settings.pool.IdleTimeout = values.GetSeconds(
		URL_PARAM_POOL_IDLE_TIMEOUT,
		DEFAULT_POOL_IDLE_TIMEOUT)

	settings.expiration = values.GetSeconds(
		URL_PARAM_EXPIRATION,
		DEFAULT_EXPIRATION)

	settings.prefix = values.GetString(
		URL_PARAM_PREFIX,
		DEFAULT_PREFIX)
	return &settings, nil
}

func NewRedisSettingsFromQuery(query string) (*RedisSettings, error) {
	values, err := godsn.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	return NewRedisSettings(values)
}

// redis dsn
type RedisDSN struct {
	*godsn.DSN
	settings *RedisSettings
}

func (r *RedisDSN) Database() int {
	path := strings.TrimLeft(r.Path(), "/")
	if db, err := strconv.Atoi(path); err == nil {
		return db
	}
	return -1
}

func (r *RedisDSN) Password() string {
	if r.User() != nil {
		if pass, ok := r.User().Password(); ok {
			return pass
		}
	}
	return ""
}

func ParseRedisDSN(dsn string) (*RedisDSN, error) {
	d, err := godsn.Parse(dsn)
	if err != nil {
		return nil, err
	}
	// this returns no error
	settings, _ := NewRedisSettings(d.DSNValues)
	rd := RedisDSN{
		d,
		settings,
	}

	rd.settings = settings

	return &rd, nil
}

// returns *redis.Poool
func (r *RedisDSN) Pool() *redis.Pool {
	pool := redis.Pool{
		MaxIdle:     r.settings.pool.MaxIdle,
		MaxActive:   r.settings.pool.MaxActive,
		IdleTimeout: r.settings.pool.IdleTimeout,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", r.Host())
			if err != nil {
				return nil, err
			}
			if db := r.Database(); db > -1 {
				if _, err := conn.Do("SELECT", db); err != nil {
					return nil, err
				}
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &pool
}

// Redis cache implementation
type RedisCache struct {
	pool       *redis.Pool
	expiration time.Duration
	settings   *RedisSettings
}

// returns data from cache
func (r *RedisCache) Get(key string) ([]byte, error) {
	conn := r.pool.Get()
	defer conn.Close()

	if result, err := redis.Bytes(conn.Do("GET", r.settings.Prefixed(key))); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

// Sets to cache
func (r *RedisCache) Set(key string, value []byte, expiration ...time.Duration) error {
	conn := r.pool.Get()
	defer conn.Close()

	d := r.expiration
	if len(expiration) > 0 {
		d = expiration[0]
	}

	args := redis.Args{}.Add(r.settings.Prefixed(key)).Add(value)
	ms := int(d / time.Millisecond)

	if ms > 0 {
		args = args.Add("PX").Add(ms)
	}

	_, err := conn.Do("SET", args...)
	return err
}

// deletes from cache
func (r *RedisCache) Delete(key string) error {
	conn := r.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", r.settings.Prefixed(key))
	return err
}

// Increments key in cache
func (r *RedisCache) Incr(key string, num ...int) (int, error) {
	conn := r.pool.Get()
	defer conn.Close()

	n := 1
	if len(num) > 0 {
		n = num[0]
	}

	return redis.Int(conn.Do("INCRBY", r.settings.Prefixed(key), n))
}

// Decrements key in cache
func (r *RedisCache) Decr(key string, num ...int) (int, error) {
	conn := r.pool.Get()
	defer conn.Close()

	n := 1
	if len(num) > 0 {
		n = num[0]
	}

	return redis.Int(conn.Do("DECRBY", r.settings.Prefixed(key), n))
}

// closes cache (it's underlying connection)
func (r *RedisCache) Close() error {
	return r.pool.Close()
}
