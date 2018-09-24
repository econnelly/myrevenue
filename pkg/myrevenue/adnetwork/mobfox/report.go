package mobfox

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

type ReportResponse struct {
	Columns     []string        `json:"columns"`
	Results     [][]interface{} `json:"results"`
	Rowcount    int             `json:"rowcount"`
	Currentness []string        `json:"currentness"`
}

func (rr *ReportRequester) Initialize() error {
	if rr.TimeZone == "" {
		rr.TimeZone = "Etc/UTC"
	}

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "api-v2.mobfox.com",
		Path:   "publisher/report",
	}

	// 2018-01-01 00:00:00
	startDate := rr.StartDate.Format("2006-01-02 15:04:05")
	endDate := rr.EndDate.Format("2006-01-02 15:04:05")

	values := url.Values{}
	values.Set("apikey", rr.APIKey)
	values.Set("from", startDate)
	values.Set("to", endDate)
	//values.Add("period", "yesterday")
	values.Add("tz", rr.TimeZone)
	values.Add("group", "sub_id,inventory_id,country_code")
	values.Add("timegroup", "day")
	values.Add("totals", "total_impressions,total_served,total_requests,total_clicks,total_earnings")

	rr.reportURL = fmt.Sprintf("%v?%v", requestUrl.String(), values.Encode())

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

func (rr ReportRequester) parse(r io.ReadCloser) ([]myrevenue.Model, error) {
	result := ReportResponse{}

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	e := json.Unmarshal(body, &result)
	if e != nil {
		return nil, e
	}

	return rr.convertModel(result)
}

func (rr ReportRequester) convertModel(m ReportResponse) ([]myrevenue.Model, error) {
	headerMap := make(map[string]int, m.Rowcount)
	for i, v := range m.Columns {
		headerMap[v] = i
	}

	reportModels := make([]myrevenue.Model, m.Rowcount)
	for j, r := range m.Results {
		reportModels[j].NetworkName = rr.GetName()
		day, err := time.Parse("2006-01-02", r[headerMap["day"]].(string))
		if err != nil {
			log.Println(err)
		} else {
			reportModels[j].DateTime = day.Format("2006-01-02 15:04:05.999999")
		}
		reportModels[j].Impressions = uint64(r[headerMap["total_impressions"]].(float64))
		reportModels[j].Revenue = r[headerMap["total_earnings"]].(float64)
		reportModels[j].Requests = uint64(r[headerMap["total_requests"]].(float64))
		clicks := r[headerMap["total_clicks"]].(float64)
		imp := r[headerMap["total_impressions"]].(float64)
		reportModels[j].CTR = clicks / imp
	}

	return reportModels, nil

}

func (ReportRequester) GetName() string {
	return "MobFox"
}
