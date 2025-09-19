package main

import (
    "encoding/json"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "time"
)

func main() {
    addr := ":8080"
    if v := os.Getenv("GATEWAY_ADDR"); v != "" {
        addr = v
    }
    searchdURL := os.Getenv("SEARCHD_URL")
    if searchdURL == "" {
        searchdURL = "http://localhost:8090"
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
        setCORS(w)
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })

    mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
        setCORS(w)
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        q := r.URL.Query().Get("q")
        u, _ := url.Parse(searchdURL)
        u.Path = "/search"
        qv := u.Query()
        qv.Set("q", q)
        u.RawQuery = qv.Encode()

        resp, err := http.Get(u.String())
        if err != nil {
            http.Error(w, "upstream error", http.StatusBadGateway)
            return
        }
        defer resp.Body.Close()
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(resp.StatusCode)
        _, _ = io.Copy(w, resp.Body)
    })

    srv := &http.Server{
        Addr:              addr,
        Handler:           loggingMiddleware(mux),
        ReadHeaderTimeout: 5 * time.Second,
    }
    log.Printf("gateway listening on %s (SEARCHD_URL=%s)", addr, searchdURL)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("server error: %v", err)
    }
}

func setCORS(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
    })
}


