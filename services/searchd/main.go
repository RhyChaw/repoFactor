package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "regexp"
    "sort"
    "strings"
    "sync"
    "time"
)

type Document struct {
    ID       string `json:"id"`
    Repo     string `json:"repo"`
    Path     string `json:"path"`
    Language string `json:"language"`
    Content  string `json:"content"`
}

type SearchResult struct {
    ID       string  `json:"id"`
    Repo     string  `json:"repo"`
    Path     string  `json:"path"`
    Language string  `json:"language"`
    Snippet  string  `json:"snippet"`
    Score    float64 `json:"score"`
}

type Index struct {
    mu        sync.RWMutex
    documents map[string]Document
    inverted  map[string]map[string]int // token -> docID -> term frequency
}

func newIndex() *Index {
    return &Index{
        documents: make(map[string]Document),
        inverted:  make(map[string]map[string]int),
    }
}

var tokenRegex = regexp.MustCompile(`\w+`)

func tokenize(text string) []string {
    matches := tokenRegex.FindAllString(strings.ToLower(text), -1)
    return matches
}

func (idx *Index) addDocuments(docs []Document) {
    idx.mu.Lock()
    defer idx.mu.Unlock()
    for _, d := range docs {
        idx.documents[d.ID] = d
        tokens := tokenize(d.Content)
        seen := make(map[string]int)
        for _, t := range tokens {
            seen[t]++
        }
        for tok, tf := range seen {
            posting, ok := idx.inverted[tok]
            if !ok {
                posting = make(map[string]int)
                idx.inverted[tok] = posting
            }
            posting[d.ID] = posting[d.ID] + tf
        }
    }
}

func (idx *Index) search(query string, limit int) []SearchResult {
    tokens := tokenize(query)
    if len(tokens) == 0 {
        return nil
    }
    idx.mu.RLock()
    defer idx.mu.RUnlock()
    scores := make(map[string]float64)
    for _, tok := range tokens {
        posting := idx.inverted[tok]
        for docID, tf := range posting {
            scores[docID] += float64(tf)
        }
    }
    results := make([]SearchResult, 0, len(scores))
    for docID, score := range scores {
        d := idx.documents[docID]
        snippet := d.Content
        if len(snippet) > 200 {
            snippet = snippet[:200]
        }
        results = append(results, SearchResult{
            ID:       d.ID,
            Repo:     d.Repo,
            Path:     d.Path,
            Language: d.Language,
            Snippet:  snippet,
            Score:    score,
        })
    }
    sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
    if limit > 0 && len(results) > limit {
        results = results[:limit]
    }
    return results
}

type indexRequest struct {
    Documents []Document `json:"documents"`
}

type searchResponse struct {
    Results []SearchResult `json:"results"`
}

func main() {
    addr := ":8090"
    if v := os.Getenv("SEARCHD_ADDR"); v != "" {
        addr = v
    }
    idx := newIndex()

    // Optional autoseed via environment variable SEARCHD_SEED_JSON
    // Expected value is a JSON array of documents or a single document JSON.
    if seed := os.Getenv("SEARCHD_SEED_JSON"); seed != "" {
        type docOrDocs struct {
            Documents []Document `json:"documents"`
        }
        var wrapper docOrDocs
        // Accept either direct {documents:[...]} or a bare array/object
        // Try {documents:[...]}
        if err := json.Unmarshal([]byte(seed), &wrapper); err == nil && len(wrapper.Documents) > 0 {
            idx.addDocuments(wrapper.Documents)
            log.Printf("autoseeded %d documents from SEARCHD_SEED_JSON wrapper", len(wrapper.Documents))
        } else {
            // Try array of docs
            var docs []Document
            if err := json.Unmarshal([]byte(seed), &docs); err == nil && len(docs) > 0 {
                idx.addDocuments(docs)
                log.Printf("autoseeded %d documents from SEARCHD_SEED_JSON array", len(docs))
            } else {
                // Try single doc
                var d Document
                if err := json.Unmarshal([]byte(seed), &d); err == nil && d.ID != "" {
                    idx.addDocuments([]Document{d})
                    log.Printf("autoseeded 1 document from SEARCHD_SEED_JSON object (id=%s)", d.ID)
                } else {
                    log.Printf("failed to parse SEARCHD_SEED_JSON: %v", err)
                }
            }
        }
    } else {
        // Default demo seed if enabled via SEARCHD_SEED_DEMO=1
        if os.Getenv("SEARCHD_SEED_DEMO") == "1" {
            demo := Document{ID: "1", Repo: "demo/repo", Path: "main.py", Language: "python", Content: "import json\njson.loads(\"{}\")"}
            idx.addDocuments([]Document{demo})
            log.Printf("autoseeded demo document (id=%s)", demo.ID)
        }
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })

    mux.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        var req indexRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "bad request", http.StatusBadRequest)
            return
        }
        idx.addDocuments(req.Documents)
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]any{"indexed": len(req.Documents)})
    })

    mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
        q := r.URL.Query().Get("q")
        limit := 25
        results := idx.search(q, limit)
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(searchResponse{Results: results})
    })

    srv := &http.Server{
        Addr:              addr,
        Handler:           loggingMiddleware(mux),
        ReadHeaderTimeout: 5 * time.Second,
    }

    log.Printf("searchd listening on %s", addr)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("server error: %v", err)
    }
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
    })
}


