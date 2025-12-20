package main

import (
  "bufio"
  "log"
  "net/http"
  "os"
  "strings"
)

var (
  opencodeURL = "http://127.0.0.1:4097"
  apiToken    = os.Getenv("OPENCODE_API_TOKEN")
)

// ============================
// BLOCKLIST (GLOBAL ENDPOINTS)
// ============================
var blockedPrefixes = []string{
  "/global/event",
  "/project",
  "/path",
  "/vcs",
  "/instance",
  "/config",
  "/provider",
  "/command",
  "/find",
  "/file",
  "/experimental",
  "/lsp",
  "/formatter",
  "/mcp",
  "/agent",
  "/auth",
}

// ============================
// Helpers
// ============================
func isBlocked(path string) bool {
  for _, p := range blockedPrefixes {
    if strings.HasPrefix(path, p) {
      return true
    }
  }
  return false
}

func authorized(r *http.Request) bool {
  return r.Header.Get("Authorization") == "Bearer "+apiToken
}

// ============================
// SSE FILTER
// ============================
func sseHandler(w http.ResponseWriter, r *http.Request) {
  if !authorized(r) {
    http.Error(w, "Unauthorized", 401)
    return
  }

  sessionID := strings.TrimPrefix(r.URL.Path, "/sse/")
  if sessionID == "" {
    http.Error(w, "Missing sessionId", 400)
    return
  }

  req, _ := http.NewRequest("GET", opencodeURL+"/global/event", nil)
  req.Header.Set("Authorization", "Bearer "+apiToken)

  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    http.Error(w, "Upstream error", 502)
    return
  }
  defer resp.Body.Close()

  w.Header().Set("Content-Type", "text/event-stream")
  w.Header().Set("Cache-Control", "no-cache")
  w.Header().Set("Connection", "keep-alive")

  flusher, ok := w.(http.Flusher)
  if !ok {
    http.Error(w, "Streaming unsupported", 500)
    return
  }

  scanner := bufio.NewScanner(resp.Body)
  for scanner.Scan() {
    line := scanner.Text()

    if !strings.HasPrefix(line, "data: ") {
      continue
    }

    data := line[6:]

    // VERY defensive filter
    if strings.Contains(data, `"sessionId":"`+sessionID+`"`) {
      w.Write([]byte("data: " + data + "\n\n"))
      flusher.Flush()
    }
  }
}

// ============================
// MAIN PROXY
// ============================
func proxyHandler(w http.ResponseWriter, r *http.Request) {
  if isBlocked(r.URL.Path) {
    http.NotFound(w, r)
    return
  }

  if !authorized(r) {
    http.Error(w, "Unauthorized", 401)
    return
  }

  req, _ := http.NewRequest(
    r.Method,
    opencodeURL+r.URL.Path,
    r.Body,
  )
  req.Header = r.Header.Clone()

  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    http.Error(w, "Upstream error", 502)
    return
  }
  defer resp.Body.Close()

  for k, v := range resp.Header {
    w.Header()[k] = v
  }
  w.WriteHeader(resp.StatusCode)
  bufio.NewReader(resp.Body).WriteTo(w)
}

// ============================
// MAIN
// ============================
func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "4096"
  }

  http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
    w.Write([]byte("ok"))
  })

  http.HandleFunc("/sse/", sseHandler)
  http.HandleFunc("/", proxyHandler)

  log.Println("TinyGo proxy listening on :" + port)
  log.Fatal(http.ListenAndServe(":"+port, nil))
}
