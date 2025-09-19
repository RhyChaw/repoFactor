package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
)

type Document struct {
    ID       string `json:"id"`
    Repo     string `json:"repo"`
    Path     string `json:"path"`
    Language string `json:"language"`
    Content  string `json:"content"`
}

type indexRequest struct {
    Documents []Document `json:"documents"`
}

func main() {
    searchd := flag.String("searchd", getenv("SEARCHD_URL", "http://localhost:8090"), "searchd base url")
    repo := flag.String("repo", "example/repo", "repo name")
    flag.Parse()

    docs := []Document{
        {ID: "1", Repo: *repo, Path: "main.py", Language: "python", Content: "import json\ndata = json.loads('{}')\nprint(data)"},
        {ID: "2", Repo: *repo, Path: "main.go", Language: "go", Content: "package main\nimport (\n\t\"encoding/json\"\n)\nfunc main() {}"},
    }

    body, _ := json.Marshal(indexRequest{Documents: docs})
    url := *searchd + "/index"
    resp, err := http.Post(url, "application/json", bytes.NewReader(body))
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    b, _ := io.ReadAll(resp.Body)
    fmt.Printf("status=%d body=%s\n", resp.StatusCode, string(b))
}

func getenv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}


