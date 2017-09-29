package repository

import (
	"fmt"

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
