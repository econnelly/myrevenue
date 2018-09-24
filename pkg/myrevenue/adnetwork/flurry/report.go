package flurry

import (
	".."
	"../.."
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"time"
)

type ReportRequester struct {
	APIKey    string
	TimeZone  string
	StartDate time.Time
	EndDate   time.Time
	adnetwork.Request

	reportURL string
}

type Response struct {
	Rows []struct {
		DateTime     string  `json:"dateTime"`
		AppName      string  `json:"app|name"`
		Impressions  int     `json:"impressions"`
		RevenueInUSD float64 `json:"revenueInUSD"`
		AdsRequested int     `json:"adsRequested"`
		ECPM         float64 `json:"eCPM"`
		Ctr          float64 `json:"ctr"`
	} `json:"rows"`
}

func (rr *ReportRequester) Initialize() error {
	startDate := rr.StartDate.Format("2006-01-02")
	endDate := rr.EndDate.Format("2006-01-02")

	if startDate == endDate {
		startDate = rr.StartDate.AddDate(0, 0, -1).Format("2006-01-02")
	}

	reportURL := url.URL{
		Scheme: "https",
		Host:   "api-metrics.flurry.com",
		Path:   fmt.Sprintf("public/v1/data/publisherRecent/%v/app", "day"),
	}

	if rr.TimeZone == "" {
		rr.TimeZone = "Etc/UTC"
	}

	query := url.Values{}
	query.Set("metrics", "impressions,revenueInUSD,adsRequested,eCPM,ctr")
	query.Add("dateTime", fmt.Sprintf("%v/%v", startDate, endDate))
	query.Add("timeZone", rr.TimeZone)
	query.Add("token", rr.APIKey)

	rr.reportURL = fmt.Sprintf("%v?%v", reportURL.String(), query.Encode())

	return nil
}

func (rr *ReportRequester) Fetch() ([]myrevenue.Model, error) {
	resp, err := myrevenue.Request(rr.reportURL, nil, false)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer resp.Body.Close()
	return rr.parse(resp.Body)
}

func (rr ReportRequester) parse(reader io.ReadCloser) ([]myrevenue.Model, error) {
	result := Response{}

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	e := json.Unmarshal(body, &result)
	if e != nil {
		return nil, e
	}
	return rr.convertToReportModel(result), nil
}

func (rr ReportRequester) Error(reader io.ReadCloser, err error) {
	reader.Close()
	log.Printf("%v: %v", rr.GetName(), err)
}

func (rr ReportRequester) convertToReportModel(result Response) []myrevenue.Model {
	reports := make([]myrevenue.Model, len(result.Rows))
	for i, row := range result.Rows {
		reports[i].NetworkName = rr.GetName()
		reports[i].Impressions = uint64(row.Impressions)
		reports[i].Revenue = row.RevenueInUSD
		reports[i].Requests = uint64(row.AdsRequested)
		reports[i].CTR = row.Ctr

		day, err := time.Parse("2006-01-02 15:04:05.000-07:00", row.DateTime)
		if err != nil {
			log.Println(err)
		} else {
			reports[i].DateTime = day.AddDate(0, 0, 1).Format("2006-01-02 15:04:05.999999")
		}
	}

	return reports
}
