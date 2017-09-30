package repository

import (
	"fmt"
	"time"

	"github.com/azaky/cpbot/util"
	"github.com/garyburd/redigo/redis"
)

type Redis struct {
	pool   *redis.Pool
	prefix string
}

func NewRedis(prefix, addr string) *Redis {
	return &Redis{
		pool: &redis.Pool{
			MaxIdle:   10,
			MaxActive: 10,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", addr)
			},
		},
		prefix: prefix,
	}
}

func (r *Redis) getUserKey() string {
	return fmt.Sprintf("%s:users", r.prefix)
}

func (r *Redis) AddUser(userID string) (interface{}, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return conn.Do("SADD", r.getUserKey(), userID)
}

func (r *Redis) RemoveUser(userID string) (interface{}, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return conn.Do("SREM", r.getUserKey(), userID)
}

func (r *Redis) GetUsers() ([]string, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", r.getUserKey()))
}

func (r *Redis) AddDaily(userID string, t int) (interface{}, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return conn.Do("ZADD", r.getDailyKey(), t, userID)
}

func (r *Redis) RemoveDaily(userID string) (interface{}, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return conn.Do("ZREM", r.getDailyKey(), userID)
}

type UserTime struct {
	User string
	Time int
}

func (r *Redis) getDailyKey() string {
	return fmt.Sprintf("%s:daily", r.prefix)
}

func (r *Redis) GetDailyWithin(from, to time.Time) ([]UserTime, error) {
	ifrom := util.TimeToInt(from)
	ito := util.TimeToInt(to)
	// TODO: handle case middle of night
	var res []UserTime
	conn := r.pool.Get()
	defer conn.Close()
	reply, err := redis.Values(conn.Do("ZRANGEBYSCORE", r.getDailyKey(), ifrom, ito, "WITHSCORES"))
	if err != nil {
		return nil, err
	}
	err = redis.ScanSlice(reply, &res)
	return res, err
}

func (r *Redis) getTimezoneKey(user string) string {
	return fmt.Sprintf("%s:timezone:%s", r.prefix, user)
}

func (r *Redis) SetTimezone(user, tz string) (interface{}, error) {
	conn := r.pool.Get()
	defer conn.Close()
	return conn.Do("SET", r.getTimezoneKey(user), tz)
}

func (r *Redis) GetTimezone(user string) (*time.Location, error) {
	conn := r.pool.Get()
	defer conn.Close()
	tz, err := redis.String(conn.Do("GET", r.getTimezoneKey(user)))
	if err != nil {
		return time.UTC, err
	}
	return util.LoadLocation(tz)
}
