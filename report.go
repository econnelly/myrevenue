package myrevenue

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"
)

type Model struct {
	NetworkName string    `json:"network_id"`
	DateTime    time.Time `json:"date_time"` // ISO 8601
	Name        string    `json:"name"`
	Country     string    `json:"country"` // 2-letter country code
	App         string    `json:"app"`
	Requests    uint64    `json:"requests"`
	Impressions uint64    `json:"impressions"`
	Clicks      uint64    `json:"clicks"`
	CTR         float64   `json:"ctr"`
	Revenue     float64   `json:"revenue"`
	ECPM        float64   `json:"ecpm"`
}

func GetRequest(reportURL string, headers map[string]string, debug bool) (*http.Response, error) {

	// Build the request
	req, err := http.NewRequest(http.MethodGet, reportURL, nil)
	if err != nil {
		return nil, err
	}

	return request(req, headers, debug)

}

func PostRequest(reportURL string, headers map[string]string, data string, debug bool) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, reportURL, strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	return request(req, headers, debug)
}

func request(req *http.Request, headers map[string]string, debug bool) (*http.Response, error) {
	if headers != nil {
		for h := range headers {
			req.Header.Set(h, headers[h])
		}
	}

	if debug {
		for k, v := range headers {
			fmt.Printf("%s: %s", k, v)
		}
	}

	client := newClient()

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Network Error: ", err)
		return nil, err
	}

	if debug {
		respHeaders, err := httputil.DumpResponse(resp, false)
		if err == nil {
			fmt.Printf(string(respHeaders))
		}
	}

	return resp, nil
}

func newClient() *http.Client {
	return &http.Client{}
}

func DateRangeFromHistory(history string, tz string) (time.Time, time.Time, error) {
	var startDate time.Time
	var endDate time.Time

	loc, e := time.LoadLocation(tz)
	if e != nil {
		log.Fatalln(e)
	}

	switch history {
	case "yesterday":
		tempDate := time.Now().In(loc).AddDate(0, 0, -1)
		startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 23, 59, 59, 999999999, loc)
	case "today":
		tempDate := time.Now().In(loc)
		startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 23, 59, 59, 999999999, loc)
	case "week":
		tempDate := time.Now().In(loc).AddDate(0, 0, -7)
		startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)

		tempDate = time.Now().In(loc).AddDate(0, 0, -1)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 23, 59, 59, 999999999, loc)
	case "month-to-date":
		tempDate := time.Now().In(loc).AddDate(0, 0, -1)
		startDate = time.Date(tempDate.Year(), tempDate.Month(), 1, 0, 0, 0, 0, loc)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 23, 59, 59, 999999999, loc)
	case "last-month":
		tempDate := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), 1, 0, 0, 0, 0, loc)
		startDate = tempDate.AddDate(0, -1, 0)

		tempDate = time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), 1, 23, 59, 59, 999999999, loc)
		endDate = tempDate.AddDate(0, 0, -1)
	default:
		tempDate, err := time.Parse("2006-01-02", history)
		if err == nil {
			startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
			endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 23, 59, 59, 999999999, loc)
		} else {
			days, err := strconv.ParseInt(history, 10, 32)
			if err != nil {
				return time.Now(), time.Now(), err
			}

			tempDate := time.Now().In(loc).AddDate(0, 0, int(days*-1))
			startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
			endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 23, 59, 59, 9999, loc)
		}
	}

	return startDate, endDate, nil
}
