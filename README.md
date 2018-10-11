## My Revenue

Every ad network has a different authentication type and response format. This project is an attempt to create a standard output for use by anyone who earns income through ad revenue. Future plans include integration into the Play Store and iTunes for app sales revenue.

Example:
```go
mopubRequest := mopub.ReportRequester {
        APIKey:     "my-api-key",
    	ReportKey:  "report-key",
    	StartDate:  "2018-09-20",
    	EndDate:    "2018-09-20",
}

// do any necessary steps to prepare for calling the reporting API
mopubRequest.Initialize()

// Fetch revenue
revenue, err := mopubRequest.Fetch()
```

Since this library attempts to standardize responses, it can only return a small subset of commonly available data. Any network-specific information can still be accessed, but the standard report is limited.
