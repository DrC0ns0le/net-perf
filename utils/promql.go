package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	prometheusAddr = "http://10.1.1.109:8428"
)

// QueryResponse represents the structure of the JSON response
type QueryPathResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Metric struct {
				Version string `json:"version"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// QueryMetrics sends a query to Prometheus and returns the parsed response.
func QueryMetrics(query string) (*QueryPathResponse, error) {

	baseURL := prometheusAddr + "/api/v1/query"
	params := url.Values{}
	params.Add("query", query)
	fullURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("error making the request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading the response: %v", err)
	}

	var response QueryPathResponse

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("non success status: %s", response.Status)
	}

	return &response, nil

}
