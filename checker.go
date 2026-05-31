package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type CheckResult struct {
	Name    string
	URL     string
	Up      bool
	Warn    bool
	Code    int
	Latency time.Duration
	Err     string
}

func checkServer(s Server) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name: s.Name,
		URL:  s.URL,
	}

	if strings.HasPrefix(s.URL, "tcp://") {
		addr := strings.TrimPrefix(s.URL, "tcp://")
		conn, err := net.DialTimeout("tcp", addr, s.timeout())
		latency := time.Since(start)
		if err != nil {
			result.Err = err.Error()
			return result
		}
		conn.Close()
		result.Up = true
		result.Latency = latency
		return result
	}



	client := &http.Client{
		Timeout: s.timeout(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse 
	},
	}

	resp, err := client.Get(s.URL)
	result.Latency = time.Since(start)
	if err != nil {
        result.Err   = shortErr(err.Error())
        result.Up    = false
        result.Warn  = false
        return result
    }
	defer resp.Body.Close()

	result.Code = resp.StatusCode
	switch {
	case resp.StatusCode >= 500:
		result.Up = false
		result.Warn = false
	case resp.StatusCode >= 400:
		result.Up = false
		result.Warn = true
	case resp.StatusCode >= 300:
        result.Up = true
        result.Warn = true
	default:
		result.Up = true
		result.Warn = false
	}
	return result
}

func shortErr(e string) string {
	if len(e) > 50 {
		return e[:50] + "…"
	}
	return e
}

func (r CheckResult) statusText() string {
	switch {
	case r.Up && !r.Warn:
		return "●  UP"
		
	case r.Up && r.Warn:
		return "»  RDR"
		
	case !r.Up && r.Warn:
		return "◆  WARN"
		
	default:
		return "⬢  DOWN"
	}
}

func (r CheckResult) codeText() string {
	if r.Code == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", r.Code)
}

func (r CheckResult) latencyText() string {
	if r.Code == 0 {
		return "—"
	}
	
	if r.Latency < time.Millisecond {
		return fmt.Sprintf("%dµs", r.Latency.Microseconds())
	}
	return fmt.Sprintf("%dms", r.Latency.Milliseconds())
}