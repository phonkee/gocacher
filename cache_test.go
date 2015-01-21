package gocacher

import (
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCache(t *testing.T) {
	Convey("Test register cache driver.", t, func() {
		driver := &RedisDriver{}
		So(func() { Register("new", driver) }, ShouldNotPanic)
		So(func() { Register("new", driver) }, ShouldPanic)
		So(func() { Register("bad", nil) }, ShouldPanic)
		So(Drivers(), ShouldContain, "new")
	})

	Convey("Test Open cache.", t, func() {
		_, err := Open("nonexisting://")
		So(err, ShouldNotBeNil)

		_, errOC := OpenConnection("redis", nil)
		So(errOC, ShouldNotBeNil)

		_, err = Open("%://")
		So(err, ShouldNotBeNil)

		_, err = OpenConnection("imap", nil)
		So(err, ShouldNotBeNil)

		pool := &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", "localhost:5379")
			},
		}

		_, err = OpenConnection("redis", pool, "expiration=100")
		So(err, ShouldBeNil)
	})
}

func TestCacheCommands(t *testing.T) {

	dsns := []string{
		"redis://localhost:6379/4",
	}

	Convey("Test cache set/get.", t, func() {
		for _, dsn := range dsns {

			testKey := "key"
			testMessage := []byte("test message")

			cache, err := Open(dsn)
			So(err, ShouldBeNil)

			errSet := cache.Set(testKey, testMessage)
			So(errSet, ShouldBeNil)

			value, errGet := cache.Get(testKey)
			So(errGet, ShouldBeNil)
			So(value, ShouldResemble, testMessage)

			_, errGetNonExisting := cache.Get("non_existing")
			So(errGetNonExisting, ShouldNotBeNil)

			testExpiryKey := "test-expiry-key"

			errSet = cache.Set(testExpiryKey, testMessage, time.Second*1)
			So(errSet, ShouldBeNil)

			time.Sleep(time.Second * 1)

			_, errGet = cache.Get(testExpiryKey)
			So(errGet, ShouldNotBeNil)

		}
	})

	Convey("Test cache delete.", t, func() {
		for _, dsn := range dsns {
			cache, errOpen := Open(dsn)
			So(errOpen, ShouldBeNil)

			testKey := "key-delete"
			testMessage := []byte("test message")

			_, err := cache.Get(testKey)
			So(err, ShouldNotBeNil)

			errSet := cache.Set(testKey, testMessage)
			So(errSet, ShouldBeNil)

			_, errGet := cache.Get(testKey)
			So(errGet, ShouldBeNil)

			errDelete := cache.Delete(testKey)
			So(errDelete, ShouldBeNil)

			_, errGetDeleted := cache.Get(testKey)
			So(errGetDeleted, ShouldNotBeNil)

			// close cache
			So(cache.Close(), ShouldBeNil)
		}
	})

	Convey("Test cache incr/decr.", t, func() {
		for _, dsn := range dsns {
			cache, errOpen := Open(dsn)
			So(errOpen, ShouldBeNil)

			testKey := "incr_key"

			// delete before test
			_ = cache.Delete(testKey)

			i, errIncr := cache.Incr(testKey)
			So(errIncr, ShouldBeNil)
			So(i, ShouldEqual, 1)

			i, errIncr = cache.Incr(testKey, 10)
			So(errIncr, ShouldBeNil)
			So(i, ShouldEqual, 11)

			var errDecr error
			i, errDecr = cache.Decr(testKey)
			So(errDecr, ShouldBeNil)
			So(i, ShouldEqual, 10)

			i, errDecr = cache.Decr(testKey, 5)
			So(errDecr, ShouldBeNil)
			So(i, ShouldEqual, 5)

			// close cache
			So(cache.Close(), ShouldBeNil)
		}
	})

}
