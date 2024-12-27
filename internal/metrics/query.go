package metrics

import (
	"context"
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
				Source  string `json:"source"`
				Target  string `json:"target"`
				Version string `json:"version"`
			} `json:"metric"`
			Value  []interface{}   `json:"value"`
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// Query sends a query to Prometheus and returns the parsed response.
func Query(ctx context.Context, query string) (*QueryPathResponse, error) {

	baseURL := prometheusAddr + "/api/v1/query"
	params := url.Values{}
	params.Add("query", query)
	fullURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error making the request: %w", err)
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making the request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading the response: %w", err)
	}

	var response QueryPathResponse

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("non success status: %s", response.Status)
	}

	return &response, nil

}

func QueryRange(ctx context.Context, start, end, step, query string) (*QueryPathResponse, error) {

	baseURL := prometheusAddr + "/api/v1/query_range"
	params := url.Values{}
	params.Add("start", start)
	params.Add("end", end)
	params.Add("step", step)
	params.Add("query", query)
	fullURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error making the request: %w", err)
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making the request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading the response: %w", err)
	}

	var response QueryPathResponse

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("non success status: %s", response.Status)
	}

	return &response, nil

}
