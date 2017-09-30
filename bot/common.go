package bot

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/azaky/cpbot/clist"
)

func generateUpcomingContestsMessage(clistService *clist.Service, startFrom, startTo time.Time, tz *time.Location, message string, limit int) ([]string, error) {
	contests, err := clistService.GetContestsStartingBetween(startFrom, startTo)
	if err != nil {
		log.Printf("Error generate24HUpcomingContestsMessage: %s", err.Error())
		return nil, err
	}

	var buffer bytes.Buffer
	buffer.WriteString(message)
	buffer.WriteString("\n")
	var res []string
	for _, contest := range contests {
		str := fmt.Sprintf("- %s. Starts at %s. Link: %s\n", contest.Name, contest.StartDate.In(tz).Format("Jan 2 15:04 MST"), contest.Link)
		if buffer.Len()+len(str) > limit {
			res = append(res, buffer.String())
			buffer = *bytes.NewBufferString(str)
		} else {
			buffer.WriteString(str)
		}
	}
	if len(contests) == 0 {
		buffer.WriteString("0 contest found")
	}
	res = append(res, buffer.String())

	return res, nil
}

func generate24HUpcomingContestsMessage(clistService *clist.Service, tz *time.Location, limit int) ([]string, error) {
	startFrom := time.Now()
	startTo := time.Now().Add(86400 * time.Second)
	return generateUpcomingContestsMessage(clistService, startFrom, startTo, tz, "Contests in the next 24 hours:", limit)
}
