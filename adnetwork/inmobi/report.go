package inmobi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/econnelly/myrevenue"
	"github.com/econnelly/myrevenue/adnetwork"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ReportRequester struct {
	SessionID string
	AccountID string
	Username  string
	SecretKey string
	StartDate time.Time
	EndDate   time.Time

	adnetwork.Request

	reportURL string
	rawData   interface{}
}

type ReportResponse struct {
	Error     bool `json:"error"`
	ErrorList []struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"errorList"`
	RespList []struct {
		AdImpressions uint64  `json:"adImpressions"`
		AdRequests    uint64  `json:"adRequests"`
		Clicks        int     `json:"clicks"`
		Earnings      float64 `json:"earnings"`
		Date          string  `json:"date"`
	} `json:"respList"`
}

type Session struct {
	RespList []struct {
		SessionID   string      `json:"sessionId"`
		AccountID   string      `json:"accountId"`
		SubAccounts interface{} `json:"subAccounts"`
	} `json:"respList"`
	Error     bool          `json:"error"`
	ErrorList []interface{} `json:"errorList"`
}

type RequestFilter struct {
	FilterName  string `json:"filterName"`
	FilterValue string `json:"filterValue"`
	Comparator  string `json:"comparator"`
}

type RequestInfo struct {
	Metrics   []string        `json:"metrics"`
	TimeFrame string          `json:"timeFrame"`
	GroupBy   []string        `json:"groupBy"`
	FilterBy  []RequestFilter `json:"filterBy"`
}

type RequestData struct {
	ReportRequest RequestInfo `json:"reportRequest"`
}

func (rr *ReportRequester) Initialize() error {
	var err error
	rr.SessionID, rr.AccountID, err = rr.startSession()
	return err
}

func (rr *ReportRequester) startSession() (string, string, error) {
	baseUrl := "https://api.inmobi.com"
	resource := "/v1.0/generatesession/generate"

	requestUrl, _ := url.ParseRequestURI(baseUrl)
	requestUrl.Path = resource

	client := &http.Client{}
	n, err := http.NewRequest(http.MethodGet, requestUrl.String(), nil)
	if err != nil {
		log.Fatalln(err)
		return "", "", err
	}

	n.Header.Add("userName", rr.Username)
	n.Header.Add("secretKey", rr.SecretKey)

	resp, err := client.Do(n)
	if err != nil {
		return "", "", err
	}

	session, err := rr.createSessionModel(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return "", "", err
	}

	return session.RespList[0].SessionID, session.RespList[0].AccountID, nil
}

func (rr ReportRequester) createSessionModel(reader io.Reader) (Session, error) {
	result := Session{}

	body, e := ioutil.ReadAll(reader)
	if e != nil {
		return Session{}, e
	}

	e = json.Unmarshal(body, &result)
	if e != nil {
		return result, e
	}

	if result.RespList == nil || len(result.RespList) == 0 {
		return result, e
	}

	return result, nil
}

func (rr *ReportRequester) Fetch() ([]myrevenue.Model, error) {
	headers := map[string]string{
		"Accept":       "application/json; charset=utf-8",
		"Content-Type": "application/json",
		"accountId":    rr.AccountID,
		"sessionId":    rr.SessionID,
		"secretKey":    rr.SecretKey,
	}

	startDate := rr.StartDate.UTC().Format("2006-01-02")
	endDate := rr.EndDate.UTC().Format("2006-01-02")

	filter := make([]RequestFilter, 1)
	filter[0].Comparator = ">"
	filter[0].FilterName = "adImpressions"
	filter[0].FilterValue = "0"

	dataStruct := RequestData{
		ReportRequest: RequestInfo{
			Metrics:   []string{"adRequests", "adImpressions", "clicks", "earnings"},
			TimeFrame: fmt.Sprintf("%v:%v", startDate, endDate),
			GroupBy:   []string{"date"},
			FilterBy:  filter,
		},
	}

	data, err := json.Marshal(dataStruct)
	if err != nil {
		return nil, err
	}

	baseUrl := "https://api.inmobi.com"
	resource := "/v3.0/reporting/publisher"

	requestUrl, _ := url.ParseRequestURI(baseUrl)
	requestUrl.Path = resource

	resp, err := myrevenue.PostRequest(requestUrl.String(), headers, string(data), false)
	defer resp.Body.Close()

	return rr.parse(resp.Body)
}

func (rr *ReportRequester) parse(reader io.ReadCloser) ([]myrevenue.Model, error) {
	result := ReportResponse{}

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	e := json.Unmarshal(body, &result)
	if e != nil {
		return nil, e
	}

	rr.rawData = result
	return rr.convertToReportModel(result)
}

func (rr ReportRequester) convertToReportModel(response ReportResponse) ([]myrevenue.Model, error) {
	reportModels := make([]myrevenue.Model, len(response.RespList))

	if response.Error {
		responseError := response.ErrorList[0]
		return nil, errors.New(responseError.Message)
	}

	loc, e := time.LoadLocation("Etc/UTC")
	if e != nil {
		return nil, e
	}

	for i, item := range response.RespList {
		reportModels[i].Impressions = item.AdImpressions
		reportModels[i].Revenue = item.Earnings
		reportModels[i].Requests = item.AdRequests
		day, parseError := time.ParseInLocation("2006-01-02 15:04:05", item.Date, loc)
		if parseError != nil {
			return nil, parseError
		}

		reportModels[i].DateTime = day
	}

	return reportModels, nil
}

func (rr ReportRequester) Error(reader io.ReadCloser, err error) {

}

func (rr ReportRequester) GetName() string {
	return "Inmobi"
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
