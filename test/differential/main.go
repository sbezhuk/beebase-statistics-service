// Command differential compares the public Health History HTTP contract of
// inspection-service (OLD) and statistics-service (NEW). It deliberately
// contains no health calculation: both services read the same authoritative
// inspection data from the running integration stack.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
)

type scenario struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Token  string `json:"token,omitempty"`
	Status int    `json:"status"`
}

type result struct {
	Status int
	Body   any
}

func main() {
	oldBase := flag.String("old", envOr("OLD_BASE_URL", "http://inspection-service:8080"), "OLD service base URL")
	newBase := flag.String("new", envOr("NEW_BASE_URL", "http://statistics-service:8080"), "NEW service base URL")
	file := flag.String("scenarios", "scenarios.json", "scenario JSON file")
	flag.Parse()

	data, err := os.ReadFile(*file)
	fatal(err)
	var scenarios []scenario
	fatal(json.Unmarshal(data, &scenarios))

	client := &http.Client{}
	failed := 0
	for _, tc := range scenarios {
		old, err := request(client, *oldBase, tc)
		if err != nil {
			fmt.Printf("FAIL\t%s\tOLD request error: %v\n", tc.Name, err)
			failed++
			continue
		}
		newResult, err := request(client, *newBase, tc)
		if err != nil {
			fmt.Printf("FAIL\t%s\tNEW request error: %v\n", tc.Name, err)
			failed++
			continue
		}

		ok := old.Status == newResult.Status && reflect.DeepEqual(old.Body, newResult.Body)
		if tc.Status != 0 {
			ok = ok && old.Status == tc.Status && newResult.Status == tc.Status
		}
		if !ok {
			failed++
			fmt.Printf("FAIL\t%s\told_status=%d new_status=%d\nOLD=%s\nNEW=%s\n", tc.Name, old.Status, newResult.Status, compact(old.Body), compact(newResult.Body))
			continue
		}
		fmt.Printf("PASS\t%s\tstatus=%d\n", tc.Name, old.Status)
	}

	if failed != 0 {
		fmt.Printf("SUMMARY\tfailed=%d total=%d\n", failed, len(scenarios))
		os.Exit(1)
	}
	fmt.Printf("SUMMARY\tfailed=0 total=%d\n", len(scenarios))
}

func request(client *http.Client, base string, tc scenario) (result, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+tc.Path, nil)
	if err != nil {
		return result{}, err
	}
	if tc.Token != "" {
		req.Header.Set("Authorization", "Bearer "+tc.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return result{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result{}, err
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return result{}, fmt.Errorf("decode status %d: %w; body=%q", resp.StatusCode, err, body)
	}
	return result{Status: resp.StatusCode, Body: decoded}, nil
}

func compact(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
