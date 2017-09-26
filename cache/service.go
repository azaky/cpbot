package cache

import (
	"fmt"
	"log"
	"regexp"

	"github.com/garyburd/redigo/redis"
	"github.com/line/line-bot-sdk-go/linebot"
)

const (
	userKey = "users"
)

var userIDRegex = regexp.MustCompile("(\\w+):(\\w+)")

type Service struct {
	conn redis.Conn
}

func NewService(conn redis.Conn) Service {
	return Service{conn}
}

func getUserID(user *linebot.EventSource) string {
	return fmt.Sprintf("%s:%s%s%s", user.Type, user.GroupID, user.RoomID, user.UserID)
}

func parseUserID(id string) (*linebot.EventSource, error) {
	matches := userIDRegex.FindStringSubmatch(id)
	if matches == nil {
		return nil, fmt.Errorf("Invalid id format")
	}
	user := &linebot.EventSource{}
	user.Type = linebot.EventSourceType(matches[1])
	userID := matches[2]
	switch user.Type {
	case linebot.EventSourceTypeGroup:
		user.GroupID = userID
	case linebot.EventSourceTypeRoom:
		user.RoomID = userID
	case linebot.EventSourceTypeUser:
		user.UserID = userID
	default:
		return nil, fmt.Errorf("Invalid userType: %s", user.Type)
	}
	return user, nil
}

func (s *Service) AddUser(user *linebot.EventSource) (interface{}, error) {
	return s.conn.Do("SADD", userKey, getUserID(user))
}

func (s *Service) RemoveUser(user *linebot.EventSource) (interface{}, error) {
	return s.conn.Do("SREM", userKey, getUserID(user))
}

func (s *Service) GetUsers() ([]*linebot.EventSource, error) {
	ids, err := redis.Strings(s.conn.Do("SMEMBERS", userKey))
	if err != nil {
		return nil, err
	}
	var users []*linebot.EventSource
	for _, id := range ids {
		user, err := parseUserID(id)
		if err != nil {
			log.Printf("cache.Service#GetUsers invalid user (%s): %s", id, err.Error())
			continue
		}
		users = append(users, user)
	}
	return users, nil
}
