package main

import (
  "bufio"
  "io"
  "log"
  "net/http"
  "os"
  "strings"
  "time"
)

var (
  opencodeURL = "http://127.0.0.1:4097"
  apiToken    = os.Getenv("OPENCODE_API_TOKEN")
)

// ============================
// BLOCKLIST (GLOBAL ENDPOINTS)
// ============================
var blockedPrefixes = []string{
  "/global/event", // HARUS lewat /sse/*
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
// HELPERS
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
// SSE HANDLER (SSE → SSE)
// ============================
func sseHandler(w http.ResponseWriter, r *http.Request) {
  if !authorized(r) {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
  }

  sessionID := strings.TrimPrefix(r.URL.Path, "/sse/")
  if sessionID == "" {
    http.Error(w, "Missing sessionId", http.StatusBadRequest)
    return
  }

  req, err := http.NewRequest("GET", opencodeURL+"/global/event", nil)
  if err != nil {
    http.Error(w, "Bad request", 400)
    return
  }

  req.Header.Set("Authorization", "Bearer "+apiToken)
  req.Header.Set("Accept", "text/event-stream")

  // IMPORTANT: no timeout for SSE
  client := &http.Client{
    Timeout: 0,
  }

  resp, err := client.Do(req)
  if err != nil {
    http.Error(w, "Upstream SSE error", 502)
    return
  }
  defer resp.Body.Close()

  // SSE headers to client
  w.Header().Set("Content-Type", "text/event-stream")
  w.Header().Set("Cache-Control", "no-cache")
  w.Header().Set("Connection", "keep-alive")
  w.WriteHeader(http.StatusOK)

  flusher, ok := w.(http.Flusher)
  if !ok {
    http.Error(w, "Streaming unsupported", 500)
    return
  }

  reader := bufio.NewReader(resp.Body)
  heartbeat := time.NewTicker(25 * time.Second)
  defer heartbeat.Stop()

  for {
    select {
    case <-r.Context().Done():
      return // client disconnected

    case <-heartbeat.C:
      // keep-alive comment
      w.Write([]byte(": ping\n\n"))
      flusher.Flush()

    default:
      line, err := reader.ReadString('\n')
      if err != nil {
        if err == io.EOF {
          return
        }
        return
      }

      log.Println(line)

      // Forward only matching session
      if strings.HasPrefix(line, "data: ") &&
        strings.Contains(line, `"sessionID":"`+sessionID+`"`) {

        w.Write([]byte(line))
        flusher.Flush()
      }
    }
  }
}

// ============================
// REST PROXY (NON-SSE)
// ============================
func proxyHandler(w http.ResponseWriter, r *http.Request) {
  if isBlocked(r.URL.Path) {
    http.NotFound(w, r)
    return
  }

  if !authorized(r) {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
  }

  req, err := http.NewRequest(
    r.Method,
    opencodeURL+r.URL.Path,
    r.Body,
  )
  if err != nil {
    http.Error(w, "Bad request", 400)
    return
  }

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
  io.Copy(w, resp.Body)
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

  // SSE endpoint
  http.HandleFunc("/sse/", sseHandler)

  // REST proxy
  http.HandleFunc("/", proxyHandler)

  log.Println("Go SSE proxy listening on :" + port)
  log.Fatal(http.ListenAndServe(":"+port, nil))
}
