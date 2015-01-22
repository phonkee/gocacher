package gocacher

import (
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRedisSettings(t *testing.T) {

	Convey("Test open/openconnection", t, func() {
		driver := &RedisDriver{}
		_, err := driver.Open("%://")
		So(err, ShouldNotBeNil)

		pool := &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", "localhost:5379")
			},
		}

		_, errOpen := driver.OpenConnection(pool, "expiration=10000")
		So(errOpen, ShouldBeNil)

		_, errOpen = driver.OpenConnection(pool, "%")
		So(errOpen, ShouldNotBeNil)

		_, errOpen = driver.OpenConnection(pool)
		So(errOpen, ShouldBeNil)
	})

	Convey("Test parse query settings", t, func() {
		var s *RedisSettings
		var err error

		// settings
		s, err = NewRedisSettingsFromQuery("pool_max_active=100&pool_max_idle=200&pool_idle_timeout=555&expiration=456&prefix=gocacher")
		So(err, ShouldBeNil)
		So(s.pool.MaxActive, ShouldEqual, 100)
		So(s.pool.MaxIdle, ShouldEqual, 200)
		So(s.pool.IdleTimeout, ShouldEqual, 555*time.Second)
		So(s.expiration, ShouldEqual, time.Second*456)
		So(s.Prefixed("str"), ShouldEqual, "gocacher:str")

		// default settings
		s, err = NewRedisSettingsFromQuery("")
		So(err, ShouldBeNil)
		So(s.pool.IdleTimeout, ShouldEqual, DEFAULT_POOL_IDLE_TIMEOUT)
		So(s.pool.MaxActive, ShouldEqual, DEFAULT_POOL_MAX_ACTIVE)
		So(s.pool.MaxIdle, ShouldEqual, DEFAULT_POOL_MAX_IDLE)
		So(s.expiration, ShouldEqual, DEFAULT_EXPIRATION)

	})

	Convey("Parse redis dsn", t, func() {
		var (
			p   *RedisDSN
			err error
		)
		p, err = ParseRedisDSN("redis://guest:pass@localhost:6379/0")
		So(err, ShouldBeNil)
		So(p.Database(), ShouldEqual, 0)
		So(p.Password(), ShouldEqual, "pass")
		So(p.Host(), ShouldEqual, "localhost:6379")

		_, err = ParseRedisDSN("%://")
		So(err, ShouldNotBeNil)

		p, _ = ParseRedisDSN("redis://localhost:63/baddb")
		So(p.Database(), ShouldEqual, -1)
		So(p.Password(), ShouldEqual, "")

		pool := p.Pool()
		conn := pool.Get()

		_, err = conn.Do("PING")
		So(err, ShouldNotBeNil)

		p, _ = ParseRedisDSN("redis://localhost:6379/123123123")
		pool = p.Pool()
		conn = pool.Get()

		_, err = conn.Do("PING")
		So(err, ShouldNotBeNil)

	})

}
