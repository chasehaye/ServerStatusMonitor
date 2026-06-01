package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
		result.Latency = time.Since(start)
		if err != nil {
			result.Err = err.Error()
		} else {
			conn.Close()
			result.Up = true
		}
		if elapsed := time.Since(start); elapsed < 2*time.Second {
			time.Sleep(2*time.Second - elapsed)
		}
		return result
	}

	if strings.HasPrefix(s.URL, "gh-actions://") {
		return checkGitHubActions(s)
	}

	if strings.HasPrefix(s.URL, "http://") || strings.HasPrefix(s.URL, "https://") {
		client := &http.Client{
			Timeout: s.timeout(),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.Get(s.URL)
		result.Latency = time.Since(start)
		if err != nil {
			result.Err  = shortErr(err.Error())
			result.Up   = false
			result.Warn = false
			if elapsed := time.Since(start); elapsed < 2*time.Second {
				time.Sleep(2*time.Second - elapsed)
			}
			return result
		}
		defer resp.Body.Close()

		result.Code = resp.StatusCode
		switch {
		case resp.StatusCode >= 500:
			result.Up   = false
			result.Warn = false
		case resp.StatusCode >= 400:
			result.Up   = false
			result.Warn = true
		case resp.StatusCode >= 300:
			result.Up   = true
			result.Warn = true
		default:
			result.Up   = true
			result.Warn = false
		}

		elapsed := time.Since(start)
		if elapsed < 2*time.Second {
			time.Sleep(2*time.Second - elapsed)
		}
		return result
	}

	log.Fatalf("unknown URL scheme for %q: must be tcp://, gh-actions://, http://, or https://", s.URL)
	return result
}

func checkGitHubActions(s Server) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name: s.Name,
		URL:  s.URL,
	}

	repoPath := strings.TrimPrefix(s.URL, "gh-actions://")

	apiURL := ""
	parts := strings.SplitN(repoPath, "/", 3)
	if len(parts) == 3 {
		branch := strings.TrimPrefix(parts[2], "refs/heads/")
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs?per_page=1&branch=%s", parts[0], parts[1], branch)
	} else {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/actions/runs?per_page=1", repoPath)
	}

	client := &http.Client{Timeout: s.timeout()}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		result.Err = shortErr(err.Error())
		return result
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	result.Latency = time.Since(start)
	if err != nil {
		result.Err = shortErr(err.Error())
		return result
	}
	defer resp.Body.Close()

	result.Code = resp.StatusCode
	if resp.StatusCode != 200 {
		result.Err = fmt.Sprintf("GitHub API returned %d", resp.StatusCode)
		return result
	}

	var payload struct {
		Runs []struct {
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
		} `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		result.Err = shortErr(err.Error())
		return result
	}

	if len(payload.Runs) == 0 {
		result.Up   = false
		result.Warn = true
		return result
	}

	run := payload.Runs[0]
	switch {
	case run.Status == "in_progress" || run.Status == "queued":
		result.Up   = true
		result.Warn = true
	case run.Conclusion == "success":
		result.Up   = true
		result.Warn = false
	case run.Conclusion == "failure":
		result.Up   = false
		result.Warn = false
	default:
		result.Up   = false
		result.Warn = true
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