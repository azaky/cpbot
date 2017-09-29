package repository

import (
	"fmt"
	"time"

	"github.com/azaky/cpbot/util"
	"github.com/garyburd/redigo/redis"
)

type Redis struct {
	conn   redis.Conn
	prefix string
}

func NewRedis(prefix string, conn redis.Conn) *Redis {
	return &Redis{
		conn:   conn,
		prefix: prefix,
	}
}

func (r *Redis) getUserKey() string {
	return fmt.Sprintf("%s:users", r.prefix)
}

func (r *Redis) AddUser(userID string) (interface{}, error) {
	return r.conn.Do("SADD", r.getUserKey(), userID)
}

func (r *Redis) RemoveUser(userID string) (interface{}, error) {
	return r.conn.Do("SREM", r.getUserKey(), userID)
}

func (r *Redis) GetUsers() ([]string, error) {
	return redis.Strings(r.conn.Do("SMEMBERS", r.getUserKey()))
}

func (r *Redis) AddDaily(userID string, t int) (interface{}, error) {
	return r.conn.Do("ZADD", r.getDailyKey(), t, userID)
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
	reply, err := redis.Values(r.conn.Do("ZRANGEBYSCORE", r.getDailyKey(), ifrom, ito, "WITHSCORES"))
	if err != nil {
		return nil, err
	}
	err = redis.ScanSlice(reply, &res)
	return res, err
}
