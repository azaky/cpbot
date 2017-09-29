package clist

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	apiURL     = "https://clist.by/api/v1/contest/"
	timeFormat = "2006-01-02T15:04:05"
)

type Contest struct {
	StartDate time.Time
	EndDate   time.Time
	Duration  time.Duration
	Name      string
	Link      string
	ID        string
}

type contestFormat struct {
	StartDate string `json:"start"`
	EndDate   string `json:"end"`
	Duration  int    `json:"duration"`
	Name      string `json:"event"`
	Link      string `json:"href"`
	ID        int    `json:"id"`
}

func (c *Contest) UnmarshalJSON(input []byte) error {
	obj := new(contestFormat)
	err := json.Unmarshal(input, obj)
	if err != nil {
		return err
	}
	c.StartDate, err = time.Parse(timeFormat, obj.StartDate)
	if err != nil {
		return err
	}
	c.EndDate, err = time.Parse(timeFormat, obj.EndDate)
	if err != nil {
		return err
	}
	c.Duration = time.Duration(obj.Duration) * time.Second
	c.Name = obj.Name
	c.Link = obj.Link
	c.ID = strconv.Itoa(obj.ID)
	return nil
}

type responseObject struct {
	Objects []Contest `json:"objects"`
}

type Service struct {
	ApiKey     string
	httpClient *http.Client
}

func NewService(apiKey string, httpClient *http.Client) *Service {
	return &Service{
		ApiKey:     apiKey,
		httpClient: httpClient,
	}
}

func (s *Service) getAuthorizationHeader() string {
	return fmt.Sprintf("ApiKey %s", s.ApiKey)
}

func (s *Service) getContests(params map[string]string) ([]Contest, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", s.getAuthorizationHeader())
	if params != nil {
		q := req.URL.Query()
		for key, value := range params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var obj responseObject
	err = json.Unmarshal(body, &obj)
	if err != nil {
		return nil, err
	}

	return obj.Objects, nil
}

func (s *Service) GetAllContests() ([]Contest, error) {
	return s.getContests(nil)
}

func (s *Service) GetContestsStartingBetween(begin, end time.Time) ([]Contest, error) {
	params := map[string]string{
		"start__lte": end.Format(timeFormat),
		"start__gte": begin.Format(timeFormat),
	}
	return s.getContests(params)
}
