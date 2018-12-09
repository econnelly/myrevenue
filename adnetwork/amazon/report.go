package amazon

import (
	"encoding/csv"
	"github.com/econnelly/myrevenue"
	"github.com/econnelly/myrevenue/adnetwork"
	"github.com/pkg/errors"
	"io"
	"log"
	"strconv"
	"time"
)

type ReportParser struct {
	adnetwork.DirectlyParsable
}

type ReportResponse struct {
	Data [][]string
}

func (r ReportParser) ParseRevenue(reader io.Reader) ([]myrevenue.Model, error) {
	models := make([]myrevenue.Model, 0)
	headerMap := make(map[string]int, 12)
	ch := make(chan []string)
	if reader == nil {
		return nil, errors.New("reader is nil")
	}

	go func() {
		r := csv.NewReader(reader)
		r.FieldsPerRecord = -1
		r.TrimLeadingSpace = true

		// Skip the first 4 lines of Amazon's CSV
		r.Read()
		r.Read()
		r.Read()

		headers, err := r.Read()
		if err != nil { //read header
			log.Printf("fatal: %v", err)
			ch <- nil
			return
		}
		for index, header := range headers {
			// Ad Earnings and eCPM headers contain currency type, but we don't want mixed currency in our revenue data

			//var key string
			//if strings.Contains(header, "Ad Earnings") {
			//	key = "Earnings"
			//} else if strings.Contains(header, "eCPM") {
			//	key = "eCPM"
			//} else {
			//	key = header
			//}
			headerMap[header] = index
		}
		defer close(ch)
		for {
			rec, err := r.Read()
			if err != nil {
				if err == io.EOF {
					ch <- nil
					break
				}
				log.Fatal(err)

			}
			ch <- rec
		}
	}()

	for {
		line := <-ch
		if line == nil {
			break
		} else {
			model, err := stringArrayToModel(headerMap, line)
			if err == nil {
				models = append(models, model)
			} else {
				return models, err
			}
		}
	}

	return models, nil
}

func stringArrayToModel(headers map[string]int, revenues []string) (myrevenue.Model, error) {
	revenue := myrevenue.Model{}
	if len(headers) != len(revenues) {
		log.Println(headers)
		log.Println(revenues)
		return revenue, errors.New("header size must match revenues size")
	}

	loc, err := time.LoadLocation("Etc/UTC")
	if err != nil {
		return revenue, err
	}

	/*
		Date	Title	Size	Region	Device	Requests	Impressions	Fill Rate %	Clicks	CTR %	eCPM (USD)	Ad Earnings (USD)
	*/

	day, err := time.ParseInLocation("2006-01-02", revenues[headers["Date"]], loc)
	revenue.DateTime = day
	revenue.Country = revenues[headers["Region"]]

	requests, err := strconv.ParseInt(revenues[headers["Requests"]], 10, 64)
	if err != nil {
		return revenue, err
	}
	revenue.Requests = uint64(requests)

	impressions, err := strconv.ParseInt(revenues[headers["Impressions"]], 10, 64)
	if err != nil {
		return revenue, err
	}
	revenue.Impressions = uint64(impressions)

	earnings, err := strconv.ParseFloat(revenues[headers["Ad Earnings (USD)"]], 64)
	if err != nil {
		return revenue, err
	}
	revenue.Revenue = float64(earnings)

	revenue.NetworkName = "Amazon"

	return revenue, nil
}
