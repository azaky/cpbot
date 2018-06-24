package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	timeRegex = regexp.MustCompile("(\\d+)(?::(\\d+)(?::(\\d+))?)?")
)

func ParseTimeInLocation(t string, loc *time.Location) (int, error) {
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
	if len(matches) > 2 && len(matches[2]) > 0 {
		m, err = strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			return -1, err
		}
		if m < 0 || m > 59 {
			return -1, fmt.Errorf("Invalid time: MM must be in range [0, 59]")
		}
	}
	if len(matches) > 3 && len(matches[3]) > 0 {
		s, err = strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			return -1, err
		}
		if s < 0 || s > 59 {
			return -1, fmt.Errorf("Invalid time: SS must be in range [0, 59]")
		}
	}
	utc := time.Date(2017, 1, 1, int(h), int(m), int(s), 0, loc).In(time.UTC)
	return utc.Hour()*3600 + utc.Minute()*60 + utc.Second(), nil
}

func ParseTime(t string) (int, error) {
	return ParseTimeInLocation(t, time.UTC)
}

func NextTime(t int) time.Time {
	h := t / 3600
	m := (t % 3600) / 60
	s := t % 60
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, time.UTC)
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func TimeToInt(t time.Time) int {
	return 3600*t.Hour() + 60*t.Minute() + t.Second()
}

func LoadLocation(tz string) (*time.Location, error) {
	// parse "UTC+x"
	if strings.HasPrefix(tz, "UTC") {
		return time.LoadLocation(strings.Replace("UTC", tz, "GMT", 1))
	}

	return time.LoadLocation(tz)
}
