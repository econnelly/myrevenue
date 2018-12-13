package mopub

import (
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/econnelly/myrevenue"
	"github.com/econnelly/myrevenue/adnetwork"
	"io"
	"net/url"
	"strconv"
	"time"
)

type ReportRequester struct {
	APIKey    string `json:"api_key"`
	ReportKey string `json:"report_key"`
	StartDate time.Time
	EndDate   time.Time
	adnetwork.Request

	reportURL string
	rawData   ReportResponse
}

type ReportResponse struct {
	Data [][]string
}

func (rr *ReportRequester) Initialize() error {
	// MoPub only allows fetching of one day at a time through the API
	// Multiple day reports need to be generated from mopub.com
	date := rr.EndDate.Format("2006-01-02")

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "app.mopub.com",
		Path:   "reports/custom/api/download_report",
	}

	query := url.Values{}
	query.Set("report_key", rr.ReportKey)
	query.Add("api_key", rr.APIKey)
	query.Add("date", date)

	rr.reportURL = fmt.Sprintf("%v?%v", requestUrl.String(), query.Encode())

	return nil
}

func (rr ReportRequester) Fetch() ([]myrevenue.Model, error) {
	resp, err := myrevenue.GetRequest(rr.reportURL, nil, false)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return rr.parse(resp.Body)
}

func (rr *ReportRequester) parse(reader io.ReadCloser) ([]myrevenue.Model, error) {
	content := csv.NewReader(reader)
	records, err := content.ReadAll()

	if err != nil {
		return nil, err
	}

	rr.rawData = ReportResponse{
		Data: records,
	}
	return rr.convertCSVToModel(records)
}

func (rr ReportRequester) convertCSVToModel(csv [][]string) ([]myrevenue.Model, error) {
	headerMap := make(map[string]int)
	csvLength := len(csv)
	if csvLength == 1 {
		return nil, errors.New(csv[0][0])
	} else if csvLength == 0 {
		return nil, errors.New("0-length csv")
	}
	reportModels := make([]myrevenue.Model, csvLength)

	loc, e := time.LoadLocation("Etc/UTC")
	if e != nil {
		return nil, e
	}

	for i := range csv {
		if i == 0 {
			for k, h := range csv[i] {
				headerMap[h] = k
			}
		} else {
			model := myrevenue.Model{}
			model.NetworkName = rr.GetName()

			model.Country = csv[i][headerMap["Country"]]

			day, err := time.ParseInLocation("2006-01-02", csv[i][headerMap["Day"]], loc)
			if err != nil {
				return nil, err
			} else {
				model.DateTime = day
			}

			ctrStr := csv[i][headerMap["CTR"]]
			if len(ctrStr) > 0 {
				ctr, err := strconv.ParseFloat(ctrStr, 32)
				if err != nil {
					return nil, err
				}
				model.CTR = ctr
			} else {
				model.CTR = 0.0
			}

			impStr := csv[i][headerMap["Impressions"]]
			if len(impStr) > 0 {

				imp, err := strconv.ParseUint(impStr, 10, 64)
				if err != nil {
					return nil, err
				}
				model.Impressions = imp
			} else {
				model.Impressions = 0
			}

			revenueStr := csv[i][headerMap["Revenue"]]
			if len(revenueStr) > 0 {
				revenue, err := strconv.ParseFloat(revenueStr, 32)
				if err != nil {
					return nil, err
				}
				model.Revenue = revenue
			} else {
				model.Revenue = 0.0
			}

			requestStr := csv[i][headerMap["Attempts"]]
			if len(requestStr) > 0 {
				requests, err := strconv.ParseUint(requestStr, 10, 64)
				if err != nil {
					return nil, err
				}
				model.Requests = requests
			} else {
				model.Requests = 0
			}

			clicksStr := csv[i][headerMap["Clicks"]]
			if len(clicksStr) > 0 {
				clicks, err := strconv.ParseUint(clicksStr, 10, 64)
				if err != nil {
					return nil, err
				}
				model.Clicks = clicks
			} else {
				model.Requests = 0
			}

			reportModels[i-1] = model
		}
	}

	return reportModels, nil

}

func (ReportRequester) GetName() string {
	return "MoPub"
}

func (rr ReportRequester) GetReport() interface{} {
	return rr.rawData
}

func (rr ReportRequester) GetStartDate() time.Time {
	return rr.StartDate
}

func (rr ReportRequester) GetEndDate() time.Time {
	return rr.EndDate
}
