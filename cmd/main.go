package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/econnelly/myrevenue/pkg/myrevenue/adnetwork"
	"github.com/econnelly/myrevenue/pkg/myrevenue/adnetwork/admob"
	"github.com/econnelly/myrevenue/pkg/myrevenue/adnetwork/flurry"
	"github.com/econnelly/myrevenue/pkg/myrevenue/adnetwork/glispa"
	"github.com/econnelly/myrevenue/pkg/myrevenue/adnetwork/mobfox"
	"github.com/econnelly/myrevenue/pkg/myrevenue/adnetwork/mopub"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

type Config struct {
	History string `json:"history"` // potential values: "yesterday", "today", "week", "month-to-date", date in ISO 8601, number of previous days
	Network struct {
		MoPub struct {
			APIKey    string `json:"api_key"`
			ReportKey string `json:"report_key"`
		} `json:"mopub"`
		Mobfox struct {
			APIKey   string `json:"api_key"`
			TimeZone string `json:"time_zone"`
		} `json:"mobfox"`
		Flurry struct {
			APIKey   string `json:"api_key"`
			TimeZone string `json:"time_zone"`
		} `json:"flurry"`
		Glispa struct {
			PublisherID  string `json:"publisher_id"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			RefreshToken string `json:"refresh_token"`
			Username     string `json:"username"`
			Password     string `json:"password"`
		} `json:"glispa"`
		Admob struct {
			PublisherID  string `json:"publisher_id"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			RefreshToken string `json:"refresh_token"`
		} `json:"admob"`
	} `json:"network"`
}

func main() {
	configFile := flag.String("c", "config.json", "Configuration file")
	config, err := PopulateConfig(*configFile)

	if err != nil {
		log.Fatalln(err)
		return
	}

	InitNetworks(config)
}

func PopulateConfig(configFile string) (*Config, error) {
	config := &Config{}
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(contents, config)

	return config, err
}

func InitNetworks(config *Config) (time.Time, time.Time, []adnetwork.Request) {
	startDate, endDate, err := datesFromHistory(config.History)

	if err != nil {
		log.Fatalln(err)
	}

	var networks []adnetwork.Request
	if config.Network.MoPub.APIKey != "" && config.Network.MoPub.ReportKey != "" {
		mopubRequester := mopub.ReportRequester{
			APIKey:    config.Network.MoPub.APIKey,
			ReportKey: config.Network.MoPub.ReportKey,
			StartDate: startDate,
			EndDate:   endDate,
		}
		networks = append(networks, mopubRequester)
	}

	if config.Network.Mobfox.APIKey != "" {
		mobfoxRequester := mobfox.ReportRequester{
			APIKey:    config.Network.Mobfox.APIKey,
			TimeZone:  config.Network.Mobfox.TimeZone,
			StartDate: startDate,
			EndDate:   endDate,
		}
		networks = append(networks, mobfoxRequester)
	}

	if config.Network.Flurry.APIKey != "" {
		flurryRequester := flurry.ReportRequester{
			APIKey:    config.Network.Flurry.APIKey,
			TimeZone:  config.Network.Flurry.TimeZone,
			StartDate: startDate,
			EndDate:   endDate,
		}
		networks = append(networks, flurryRequester)
	}

	if config.Network.Glispa.PublisherID != "" {
		glispaRequester := glispa.ReportRequester{
			PublisherKey: config.Network.Glispa.PublisherID,
			ClientID:     config.Network.Glispa.ClientID,
			ClientSecret: config.Network.Glispa.ClientSecret,
			RefreshToken: config.Network.Glispa.RefreshToken,
			Username:     config.Network.Glispa.Username,
			Password:     config.Network.Glispa.Password,
			StartDate:    startDate,
			EndDate:      endDate,
		}
		networks = append(networks, glispaRequester)
	}

	if config.Network.Admob.PublisherID != "" {
		admobRequester := admob.ReportRequester{
			PublisherID:  config.Network.Admob.PublisherID,
			ClientID:     config.Network.Admob.ClientID,
			ClientSecret: config.Network.Admob.ClientSecret,
			RefreshToken: config.Network.Admob.RefreshToken,
			StartDate:    startDate,
			EndDate:      endDate,
		}
		networks = append(networks, admobRequester)
	}

	return startDate, endDate, networks
}

func MakeRequest(n adnetwork.Request, ch chan<- string) {
	err := n.Initialize()
	if err != nil {
		ch <- err.Error()
		return
	}

	reportModels, err := n.Fetch()
	if err != nil {
		ch <- err.Error()
		return
	}

	var revenue float64 = 0
	var impressions uint64 = 0
	var requests uint64 = 0
	for i := range reportModels {
		revenue += reportModels[i].Revenue
		impressions += reportModels[i].Impressions
		requests += reportModels[i].Requests
	}

	ch <- fmt.Sprintf("%v\n\tRevenue: $%.2f\n\tRequests: %v\n\tImpressions: %v\n", n.GetName(), revenue, requests, impressions)
}

func datesFromHistory(history string) (time.Time, time.Time, error) {
	var startDate time.Time
	var endDate time.Time

	loc, e := time.LoadLocation("UTC")
	if e != nil {
		log.Fatalln(e)
	}
	switch history {
	case "yesterday":
		tempDate := time.Now().UTC().AddDate(0, 0, -1)
		startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 11, 59, 59, 999999999, loc)
	case "today":
		tempDate := time.Now().UTC()
		startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 11, 59, 59, 999999999, loc)
	case "week":
		tempDate := time.Now().UTC().AddDate(0, 0, -7)
		startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)

		tempDate = time.Now().UTC().AddDate(0, 0, -1)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 11, 59, 59, 999999999, loc)
	case "month-to-date":
		tempDate := time.Now().UTC().AddDate(0, 0, -1)
		startDate = time.Date(tempDate.Year(), tempDate.Month(), 1, 0, 0, 0, 0, loc)
		endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 11, 59, 59, 999999999, loc)
	default:
		tempDate, err := time.Parse("2006-01-02", history)
		if err == nil {
			startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
			endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 11, 59, 59, 999999999, loc)
		} else {
			days, err := strconv.ParseInt(history, 10, 32)
			if err != nil {
				return time.Now(), time.Now(), err
			}

			tempDate := time.Now().UTC().AddDate(0, 0, int(days*-1))
			startDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 0, 0, 0, 0, loc)
			endDate = time.Date(tempDate.Year(), tempDate.Month(), tempDate.Day(), 11, 59, 59, 9999, loc)
		}
	}

	log.Printf("Revenue for %v to %v", startDate, endDate)

	return startDate, endDate, nil
}