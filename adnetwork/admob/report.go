package admob

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
	"strconv"
	"strings"
	"time"
)

type ReportRequester struct {
	PublisherID  string
	ClientID     string
	ClientSecret string
	RefreshToken string
	StartDate    time.Time
	EndDate      time.Time
	adnetwork.Request

	authToken string
	reportURL string

	rawData ReportResponse
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

type ReportResponse struct {
	Kind             string `json:"kind"`
	TotalMatchedRows string `json:"totalMatchedRows"`
	Headers          []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Currency string `json:"currency,omitempty"`
	} `json:"headers"`
	Rows      [][]string    `json:"rows"`
	Totals    []string      `json:"totals"`
	Averages  []interface{} `json:"averages"`
	StartDate string        `json:"startDate"`
	EndDate   string        `json:"endDate"`
}

type TokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type GenericErrorResponse struct {
	Error struct {
		Errors []struct {
			Domain       string `json:"domain"`
			Reason       string `json:"reason"`
			Message      string `json:"message"`
			LocationType string `json:"locationType"`
			Location     string `json:"location"`
		} `json:"errors"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const (
	EARNINGS            = "EARNINGS"
	IMPRESSIONS         = "IMPRESSIONS"
	AD_REQUESTS         = "AD_REQUESTS"
	AD_REQUESTS_CTR     = "AD_REQUESTS_CTR"
	CLICKS              = "CLICKS"
	MATCHED_AD_REQUESTS = "MATCHED_AD_REQUESTS"
)

func (rr *ReportRequester) Initialize() error {
	rr.authToken = rr.fetchAuthToken()
	if rr.authToken == "" {
		log.Fatal("Empty auth token")
		return errors.New("empty auth token")
	}

	requestUrl := url.URL{
		Scheme: "https",
		Host:   "www.googleapis.com",
		Path:   fmt.Sprintf("adsense/v1.4/accounts/%v/reports", rr.PublisherID),
	}

	startDate := fmt.Sprintf("%04d-%02d-%02d", rr.StartDate.Year(), int(rr.EndDate.Month()), rr.StartDate.Day())
	endDate := fmt.Sprintf("%04d-%02d-%02d", rr.EndDate.Year(), int(rr.EndDate.Month()), rr.EndDate.Day())

	query := url.Values{}
	query.Set("startDate", startDate)
	query.Add("endDate", endDate)
	query.Add("metric", EARNINGS)
	query.Add("metric", IMPRESSIONS)
	query.Add("metric", AD_REQUESTS)
	query.Add("metric", AD_REQUESTS_CTR)
	query.Add("metric", CLICKS)
	query.Add("metric", MATCHED_AD_REQUESTS)

	rr.reportURL = fmt.Sprintf("%v?%v", requestUrl.String(), query.Encode())

	return nil
}

func (rr *ReportRequester) Fetch() ([]myrevenue.Model, error) {
	headers := map[string]string{
		"Accept":        "application/json; charset=utf-8",
		"Authorization": fmt.Sprintf("Bearer %v", rr.authToken),
	}
	resp, err := myrevenue.GetRequest(rr.reportURL, headers, false)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer resp.Body.Close()
	return rr.parse(resp.Body)
}

func (rr ReportRequester) parse(r io.Reader) ([]myrevenue.Model, error) {
	result := ReportResponse{}

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	e := json.Unmarshal(body, &result)
	if e != nil {
		return nil, e
	}

	return rr.convertToReportModel(result)
}

func (rr ReportRequester) convertToReportModel(r ReportResponse) ([]myrevenue.Model, error) {
	headers := make(map[string]int)
	for i, h := range r.Headers {
		headers[h.Name] = i
	}

	reportModels := make([]myrevenue.Model, len(r.Rows))
	for i, result := range r.Rows {
		reportModels[i].NetworkName = rr.GetName()
		impressions, err := strconv.ParseUint(result[headers[IMPRESSIONS]], 10, 64)
		if err == nil {
			reportModels[i].Impressions = impressions
		}

		revenue, err := strconv.ParseFloat(result[headers[EARNINGS]], 64)
		if err == nil {
			reportModels[i].Revenue = revenue
		}

		requests, err := strconv.ParseUint(result[headers[AD_REQUESTS]], 10, 64)
		if err == nil {
			reportModels[i].Requests = requests
		}

		ctr, err := strconv.ParseFloat(result[headers[AD_REQUESTS_CTR]], 64)
		if err == nil {
			reportModels[i].CTR = ctr
		}

		clicks, err := strconv.ParseUint(result[headers[CLICKS]], 10, 64)
		if err == nil {
			reportModels[i].Clicks = clicks
		}

		day, err := time.Parse("2006-01-02", r.StartDate)
		if err != nil {
			log.Println(err.Error())
		} else {
			reportModels[i].DateTime = day.Format("2006-01-02 15:04:05.999999")
		}
	}

	return reportModels, nil
}

func (rr ReportRequester) fetchAuthToken() string {
	baseUrl := "https://accounts.google.com"
	resource := "/o/oauth2/token"
	body := url.Values{}
	body.Set("client_id", rr.ClientID)
	body.Add("client_secret", rr.ClientSecret)
	body.Add("grant_type", "refresh_token")
	body.Add("refresh_token", rr.RefreshToken)

	requestUrl, _ := url.ParseRequestURI(baseUrl)
	requestUrl.Path = resource

	client := &http.Client{}
	r, err := http.NewRequest(http.MethodPost, requestUrl.String(), strings.NewReader(body.Encode()))
	if err != nil {
		log.Fatalln(err)
		return ""
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := client.Do(r)
	if err != nil {
		log.Println(err)
		return ""
	}

	authModel, errorModel, err := rr.unmarshalAuth(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Println(err)
		return ""
	}

	if authModel.AccessToken == "" {
		log.Println(errorModel.ErrorDescription)
	}

	rr.RefreshToken = authModel.RefreshToken
	return authModel.AccessToken
}

func (rr ReportRequester) unmarshalAuth(r io.ReadCloser) (TokenResponse, TokenErrorResponse, error) {
	result := TokenResponse{}

	body, e := ioutil.ReadAll(r)
	if e != nil {
		return TokenResponse{}, TokenErrorResponse{}, e
	}

	e = json.Unmarshal(body, &result)
	if e != nil {
		authError := TokenErrorResponse{}
		e = json.Unmarshal(body, &authError)
		if e != nil {
			return TokenResponse{}, TokenErrorResponse{}, e
		}

		return TokenResponse{}, authError, nil
	}

	return result, TokenErrorResponse{}, nil
}

func (rr ReportRequester) GetName() string {
	return "AdMob"
}

func (rr ReportRequester) GetReport() interface{} {
	return rr.rawData
}
