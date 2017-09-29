package util

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var (
	timeRegex = regexp.MustCompile("(\\d+)(?::(\\d+)(?::(\\d+))?)?")
)

func ParseTime(t string) (int, error) {
	matches := timeRegex.FindStringSubmatch(t)
	if len(matches) == 0 {
		return -1, fmt.Errorf("Invalid time: should be in format HH[:MM[:SS]]")
	}
	h, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return -1, err
	}
	if h < 0 || h > 23 {
		return -1, fmt.Errorf("Invalid time: HH must be in range [0, 23]")
	}
	var m, s int64
	if len(matches) > 2 {
		m, err = strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			return -1, err
		}
		if m < 0 || m > 59 {
			return -1, fmt.Errorf("Invalid time: MM must be in range [0, 59]")
		}
	}
	if len(matches) > 3 {
		s, err = strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			return -1, err
		}
		if s < 0 || s > 59 {
			return -1, fmt.Errorf("Invalid time: SS must be in range [0, 59]")
		}
	}
	return int(h*3600 + m*60 + s), nil
}

func NextTime(t int) time.Time {
	h := t / 3600
	m := (t % 3600) / 60
	s := t % 60
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, now.Location())
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func TimeToInt(t time.Time) int {
	return 3600*t.Hour() + 60*t.Minute() + t.Second()
}
