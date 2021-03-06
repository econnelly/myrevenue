package glispa

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
	"strings"
	"time"
)

type ReportRequester struct {
	PublisherKey string `json:"publisher_key"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"` // This should be fetched and stored when the user connect their Glispa account to the system
	// Once that is done, this field will no longer need to be public
	Username  string `json:"username"`
	Password  string `json:"password"`
	StartDate time.Time
	EndDate   time.Time

	adnetwork.Request

	authToken string
	reportURL string
	rawData   ReportResponse
}

type ReportResponse struct {
	Query struct {
		Filters struct {
			AdunitID          []string `json:"adunit_id"`
			AdunitType        []string `json:"adunit_type"`
			AppID             []string `json:"app_id"`
			AppVersion        []string `json:"app_version"`
			Carrier           []string `json:"carrier"`
			City              []string `json:"city"`
			ConnectionType    []string `json:"connection_type"`
			Country           []string `json:"country"`
			DeviceBrand       []string `json:"device_brand"`
			DeviceModel       []string `json:"device_model"`
			DeviceOrientation []string `json:"device_orientation"`
			DeviceOsVersion   []string `json:"device_os_version"`
			DeviceOs          []string `json:"device_os"`
			PublisherID       []string `json:"publisher_id"`
			SdkVersion        []string `json:"sdk_version"`
			Gender            []string `json:"gender"`
			YearOfBirth       []string `json:"year_of_birth"`
			InterestCategory  []string `json:"interest_category"`
			Persona           []string `json:"persona"`
		} `json:"filters"`
		Granularity string `json:"granularity"`
		Timestamp   struct {
			From time.Time `json:"from"`
			To   time.Time `json:"to"`
		} `json:"timestamp"`
		Group []string `json:"group"`
	} `json:"query"`
	Data []struct {
		Timestamp  time.Time `json:"timestamp"`
		Dimensions struct {
			AdunitID          string `json:"adunit_id"`
			AdunitType        string `json:"adunit_type"`
			AppID             string `json:"app_id"`
			AppVersion        string `json:"app_version"`
			Carrier           string `json:"carrier"`
			City              string `json:"city"`
			ConnectionType    string `json:"connection_type"`
			Country           string `json:"country"`
			DeviceBrand       string `json:"device_brand"`
			DeviceModel       string `json:"device_model"`
			DeviceOrientation string `json:"device_orientation"`
			DeviceOsVersion   string `json:"device_os_version"`
			DeviceOs          string `json:"device_os"`
			Gender            string `json:"gender"`
			InterestCategory  string `json:"interest_category"`
			Persona           string `json:"persona"`
			PublisherID       string `json:"publisher_id"`
			SdkVersion        string `json:"sdk_version"`
			YearOfBirth       string `json:"year_of_birth"`
		} `json:"dimensions"`
		Result struct {
			AdRequests  int     `json:"ad_requests"`
			Clicks      int     `json:"clicks"`
			Ctr         float64 `json:"ctr"`
			Earnings    float64 `json:"earnings"`
			Ecpm        float64 `json:"ecpm"`
			Impressions int     `json:"impressions"`
			RenderRate  float64 `json:"render_rate"`
			FillRate    float64 `json:"fill_rate"`
		} `json:"result"`
	} `json:"data"`
	Meta struct {
		Currency string `json:"currency"`
	} `json:"meta"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type AuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (rr *ReportRequester) Initialize() error {
	accessToken := rr.fetchAccessToken()
	if accessToken == "" {

		if rr.hasLoginCredentials() {
			rr.RefreshToken = ""
			accessToken = rr.fetchAccessToken()
		} else {
			return errors.New("empty access token")
		}
	}

	startDate := rr.StartDate.UTC().Format("2006-01-02 15:04:05.999999999")
	endDate := rr.EndDate.UTC().Format("2006-01-02 15:04:05.999999999")

	reportUrl := url.URL{
		Scheme: "https",
		Host:   "reporting.glispaconnect.com",
		Path:   fmt.Sprintf("v1.1/publishers/%v", rr.PublisherKey),
	}

	query := url.Values{}
	query.Set("access_token", accessToken)
	query.Add("timestamp[from]", startDate)
	query.Add("timestamp[to]", endDate)
	query.Add("granularity", "hour")

	rr.reportURL = fmt.Sprintf("%v?%v", reportUrl.String(), query.Encode())

	return nil
}

func (rr *ReportRequester) Fetch() ([]myrevenue.Model, error) {
	headers := map[string]string{
		"Accept":        "application/json; charset=utf-8",
		"Authorization": fmt.Sprintf("Bearer %v", rr.authToken),
	}
	resp, err := myrevenue.GetRequest(rr.reportURL, headers, false)

	if err != nil {
		return nil, err
	}

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

func (rr ReportRequester) fetchAccessToken() string {
	baseUrl := "https://auth.glispaconnect.com"
	resource := "/token"
	body := url.Values{}
	body.Set("client_id", rr.ClientID)
	body.Add("client_secret", rr.ClientSecret)

	var grantType string
	if rr.RefreshToken != "" {
		grantType = "refresh_token"
		body.Add("refresh_token", rr.RefreshToken)
	} else {
		grantType = "password"
		body.Add("username", rr.Username)
		body.Add("password", rr.Password)
	}

	body.Add("grant_type", grantType)

	requestUrl, _ := url.ParseRequestURI(baseUrl)
	requestUrl.Path = resource

	client := &http.Client{}
	n, err := http.NewRequest(http.MethodPost, requestUrl.String(), strings.NewReader(body.Encode()))
	if err != nil {
		log.Fatalln(err)
		return ""
	}

	n.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := client.Do(n)
	if err != nil {
		return ""
	}

	authModel, _, err := rr.unmarshalAuth(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return ""
	}

	rr.RefreshToken = authModel.RefreshToken
	return authModel.AccessToken
}

func (rr ReportRequester) unmarshalAuth(reader io.Reader) (AuthResponse, AuthErrorResponse, error) {
	result := AuthResponse{}

	body, e := ioutil.ReadAll(reader)
	if e != nil {
		return AuthResponse{}, AuthErrorResponse{}, e
	}

	e = json.Unmarshal(body, &result)
	if e != nil {
		return result, AuthErrorResponse{}, e
	}

	if result.AccessToken == "" {
		authError := AuthErrorResponse{}
		e = json.Unmarshal(body, &authError)
		if e != nil {
			return AuthResponse{}, AuthErrorResponse{}, e
		}

		return result, authError, e
	}

	return result, AuthErrorResponse{}, nil
}

func (rr ReportRequester) Error(reader io.ReadCloser, err error) {
	errorResult := AuthErrorResponse{}
	body, err := ioutil.ReadAll(reader)
	defer reader.Close()
	if err != nil {
		return
	}
	e := json.Unmarshal(body, &errorResult)
	if e == nil {
		return
	}
}

func (rr ReportRequester) convertToReportModel(response ReportResponse) ([]myrevenue.Model, error) {
	reportModels := make([]myrevenue.Model, len(response.Data))

	for i, d := range response.Data {
		reportModels[i].NetworkName = rr.GetName()
		reportModels[i].Revenue = d.Result.Earnings
		reportModels[i].Impressions = uint64(d.Result.Impressions)
		reportModels[i].Requests = uint64(d.Result.AdRequests)
		reportModels[i].CTR = d.Result.Ctr
		reportModels[i].DateTime = d.Timestamp
	}

	return reportModels, nil
}

func (rr ReportRequester) hasLoginCredentials() bool {
	return rr.Username != "" && rr.Password != ""
}

func (rr ReportRequester) hasRefreshToken() bool {
	return rr.RefreshToken != ""
}

func (rr ReportRequester) GetName() string {
	return "Glispa"
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
