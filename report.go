package myrevenue

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

type Model struct {
	NetworkName string  `json:"network_id"`
	DateTime    string  `json:"date_time"` // ISO 8601
	Name        string  `json:"name"`
	Country     string  `json:"country"` // 2-letter country code
	App         string  `json:"app"`
	Requests    uint64  `json:"requests"`
	Impressions uint64  `json:"impressions"`
	Clicks      uint64  `json:"clicks"`
	CTR         float64 `json:"ctr"`
	Revenue     float64 `json:"revenue"`
	ECPM        float64 `json:"ecpm"`
}

func Request(reportURL string, headers map[string]string, debug bool) (*http.Response, error) {

	// Build the request
	req, err := http.NewRequest(http.MethodGet, reportURL, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return nil, err
	}

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
