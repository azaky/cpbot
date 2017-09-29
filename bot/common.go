package bot

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/azaky/cplinebot/clist"
)

func generateUpcomingContestsMessage(clistService *clist.Service, startFrom, startTo time.Time, message string) (string, error) {
	contests, err := clistService.GetContestsStartingBetween(startFrom, startTo)
	if err != nil {
		log.Printf("Error generate24HUpcomingContestsMessage: %s", err.Error())
		return "", err
	}

	var buffer bytes.Buffer
	buffer.WriteString(message)
	buffer.WriteString("\n")
	for _, contest := range contests {
		buffer.WriteString(fmt.Sprintf("- %s. Starts at %s. Link: %s\n", contest.Name, contest.StartDate.Format("Jan 2 15:04 MST"), contest.Link))
	}
	if len(contests) == 0 {
		buffer.WriteString("0 contest found")
	}

	return buffer.String(), nil
}

func generate24HUpcomingContestsMessage(clistService *clist.Service) (string, error) {
	startFrom := time.Now()
	startTo := time.Now().Add(86400 * time.Second)
	return generateUpcomingContestsMessage(clistService, startFrom, startTo, "Contests in the next 24 hours:")
}
