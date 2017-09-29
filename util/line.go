package util

import (
	"fmt"
	"regexp"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	lineEventSourceRegex = regexp.MustCompile("(\\w+):(\\w+)")
)

func LineEventSourceToReplyString(es *linebot.EventSource) string {
	switch es.Type {
	case linebot.EventSourceTypeGroup:
		return es.GroupID
	case linebot.EventSourceTypeRoom:
		return es.RoomID
	case linebot.EventSourceTypeUser:
		return es.UserID
	default:
		return ""
	}
}

func LineEventSourceToString(es *linebot.EventSource) string {
	return fmt.Sprintf("%s:%s", es.Type, LineEventSourceToReplyString(es))
}

func StringToLineEventSource(s string) (*linebot.EventSource, error) {
	matches := lineEventSourceRegex.FindStringSubmatch(s)
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

func StringsToLineEventSources(ss []string) ([]*linebot.EventSource, error) {
	var ess []*linebot.EventSource
	for _, s := range ss {
		if es, err := StringToLineEventSource(s); err == nil {
			ess = append(ess, es)
		}
	}
	return ess, nil
}
