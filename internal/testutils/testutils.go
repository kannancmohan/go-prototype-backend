package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func WaitForPort(port int, timeout time.Duration) error {
	start := time.Now()
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Since(start) > timeout {
			return fmt.Errorf("timed out[%v] waiting for port%d", timeout, port)
		}
		time.Sleep(100 * time.Millisecond) // Polling interval
	}
}

// RetryGetReq is a generic function that performs an HTTP GET request with retries.
func RetryGetReq(url, expectedString string, expectedStatusCode, maxRetries int, retryDelay time.Duration) (string, error) {
	client := &http.Client{}

	for attempt := 1; attempt <= maxRetries; attempt++ {

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create HTTP request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != expectedStatusCode {
			time.Sleep(retryDelay)
			continue
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			time.Sleep(retryDelay)
			continue
		}
		bodyStr := string(body)

		if expectedString != "" {
			if strings.Contains(bodyStr, expectedString) {
				return bodyStr, nil
			}
		} else if resp.StatusCode == expectedStatusCode {
			return bodyStr, nil
		}

		time.Sleep(retryDelay)
	}
	return "", fmt.Errorf("max retries reached")
}

// RetryGetReqForJson is a generic function that performs an HTTP GET request with retries.
// It checks if the JSON response contains a specific field and returns the marshaled JSON according to the generic type T.
func RetryGetReqForJson[T any](url, expectedField string, expectedStatusCode, maxRetries int, retryDelay time.Duration) (*T, error) {
	client := &http.Client{}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != expectedStatusCode {
			time.Sleep(retryDelay)
			continue
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			time.Sleep(retryDelay)
			continue
		}

		// Check if the JSON contains the expected field
		if expectedField != "" {
			if !bytes.Contains(body, []byte(`"`+expectedField+`"`)) {
				time.Sleep(retryDelay)
				continue
			}
		}

		// Unmarshal the JSON into the generic type T
		var result T
		if err := json.Unmarshal(body, &result); err != nil {
			time.Sleep(retryDelay)
			continue
		}

		return &result, nil
	}

	return nil, fmt.Errorf("max retries reached")
}

func GetFreePorts(numPorts int) ([]int, error) {
	var ports []int
	for range numPorts {
		a, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return nil, err
		}

		l, err := net.ListenTCP("tcp", a)
		if err != nil {
			return nil, err
		}
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return ports, nil
}

func GetLocalIP() (string, error) {
	_, err := os.Stat("/.dockerenv")
	if err == nil { // If running inside a Docker container, return "localhost".
		return "localhost", nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("local ip not found")
}
