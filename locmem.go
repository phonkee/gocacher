package gocacher

import (
	"errors"

	"time"

	"sync"

	"strconv"

	"strings"

	"github.com/phonkee/godsn"
)

const (
	LOCMEM_DEFAULT_DATABASE = "default"
)

var (
	ErrNotFound = errors.New("locmem item not found")

	// storage where all databases are stored
	storage map[string]map[string]*locmemCacheItem

	// mutex to lock storage for concurrent access
	mutex *sync.RWMutex
)

func init() {
	Register("locmem", &locmemDriver{})

	// Instantiate blank storage
	storage = map[string]map[string]*locmemCacheItem{}
	mutex = &sync.RWMutex{}
}

/*
locmemDriver - implementation of local memory cache.

It provides extended functionality such as garbage collector
*/
type locmemDriver struct {
}

func (l *locmemDriver) Open(dsn string) (cacher Cacher, err error) {

	if settings, errSettings := newLocmemSettings(dsn); errSettings != nil {
		err = errSettings
		return
	} else {
		cacher = newLocmemCache(settings)
	}
	return
}

func (l *locmemDriver) OpenConnection(connection interface{}, settings ...string) (cacher Cacher, err error) {
	panic("doesn't make sense for locmem cache")
	return
}

/*
locmemCache
*/

func newLocmemCache(settings *locmemSettings) Cacher {
	return &locmemCache{
		settings: settings,
	}
}

type locmemCache struct {
	settings *locmemSettings
}

/*
Get returns cache value if found
*/
func (l *locmemCache) Get(key string) (result []byte, err error) {

	hasExpired := false

	db := getDatabase(l.settings.database)

	func() {
		mutex.RLock()
		defer mutex.RUnlock()

		if item, ok := db[key]; ok {
			// cache item found
			if item.expiration.IsZero() {
				result = item.value
			} else {
				if time.Now().Before(item.expiration) {
					result = item.value
				} else {
					// delete since expired
					err = ErrNotFound
					hasExpired = true
				}
			}
		} else {
			// cache item not found
			err = ErrNotFound
		}

	}()

	// item has expired - delete it
	if hasExpired {
		l.Delete(key)
	}

	return
}

func (l *locmemCache) Set(key string, value []byte, expiration ...time.Duration) (err error) {
	db := getDatabase(l.settings.database)

	mutex.Lock()
	defer mutex.Unlock()

	// expiration value to cache item

	expval := time.Time{}

	if len(expiration) == 1 {
		expval = time.Now().Add(expiration[0])
	} else if len(expiration) > 1 {
		panic("set multiple expirations in one call")
	}

	db[key] = &locmemCacheItem{
		expiration: expval,
		value:      value,
	}

	return
}

// Deletes key in cache
func (l *locmemCache) Delete(key string) (err error) {

	db := getDatabase(l.settings.database)

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := db[key]; ok {
		// delete from cache if exists
		delete(db, key)
	} else {
		err = ErrNotFound
	}

	return
}

// Incr increments key either by 1, or if num is given by that amout will be incremented
func (l *locmemCache) Incr(key string, num ...int) (result int, err error) {
	db := getDatabase(l.settings.database)

	mutex.Lock()
	defer mutex.Unlock()

	if len(num) > 0 {
		result = num[0]
	} else {
		result = 1

	}

	expiration := time.Time{}

	if item, ok := db[key]; ok {

		if numValue, errParse := strconv.Atoi(string(item.value)); errParse == nil {
			result = numValue + result
		}
	}

	item := &locmemCacheItem{
		expiration: expiration,
		value:      []byte(strconv.Itoa(result)),
	}
	db[key] = item

	return
}

// Decrements key by 1, if num is given it decrements by given number
func (l *locmemCache) Decr(key string, num ...int) (result int, err error) {
	db := getDatabase(l.settings.database)

	mutex.Lock()
	defer mutex.Unlock()

	if len(num) > 0 {
		result = num[0]
	} else {
		result = 1

	}

	// zero expiration for incr/decr
	expiration := time.Time{}

	if item, ok := db[key]; ok {

		if numValue, errParse := strconv.Atoi(string(item.value)); errParse == nil {
			result = numValue - result
		}
	}

	item := &locmemCacheItem{
		expiration: expiration,
		value:      []byte(strconv.Itoa(result)),
	}
	db[key] = item

	return
}

// Return cache to cache pool
func (l *locmemCache) Close() (err error) {
	/*
		Here we should stop garbage collect if it was running
	*/
	return
}

/*
cache Item
*/
type locmemCacheItem struct {
	expiration time.Time
	value      []byte
}

func (l *locmemCacheItem) IsValid(t time.Time) bool {

	if l.expiration.IsZero() {
		return true
	}

	return t.Before(l.expiration)
}

/*
locmemSettings
*/
type locmemSettings struct {

	// database name
	database string

	// enable garbage collect of old cache items
	expiration time.Duration
}

func newLocmemSettings(dsn string) (settings *locmemSettings, err error) {
	settings = &locmemSettings{}

	if values, errParse := godsn.Parse(dsn); err != nil {
		return nil, errParse
	} else {

		exp_param := values.GetString(URL_PARAM_EXPIRATION, "")
		if exp_param == "" {
			settings.expiration = DEFAULT_EXPIRATION
		} else {
			if settings.expiration, err = time.ParseDuration(exp_param); err != nil {
				return
			}

		}

		// set database name
		settings.database = strings.Trim(strings.TrimSpace(values.Path()), "/")
		if settings.database == "" {
			settings.database = LOCMEM_DEFAULT_DATABASE
		}
	}

	return
}

/*
Storage utilities
*/
func getDatabase(database string) (result map[string]*locmemCacheItem) {
	mutex.Lock()
	defer mutex.Unlock()

	var ok bool

	if result, ok = storage[database]; !ok {
		storage[database] = map[string]*locmemCacheItem{}
		result = storage[database]
	}

	return
}
