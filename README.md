PolyScale: Distributed, AI-Accelerated Code Search and Refactoring Engine

Overview

PolyScale is a monorepo that hosts a minimal, end-to-end skeleton of a Google-scale code search and refactoring system. It includes:
- Gateway HTTP API (Go)
- Search daemon with in-memory index (Go)
- ML service for embeddings and refactor suggestions (Python/FastAPI, stub)
- Proto contracts (gRPC), initial draft

Status: early scaffold with runnable services locally.

Getting Started

Prerequisites
- Go 1.21+ installed
- Python 3.10+ installed

Quickstart
1) Run search daemon (port 8090):
   cd services/searchd && go run ./...

2) Run gateway (port 8080):
   SEARCHD_URL=http://localhost:8090 cd ../gateway && go run ./...

3) Run ML service (port 8000):
   cd ../../ml/service && python -m venv .venv && source .venv/bin/activate && \
   pip install -r requirements.txt && uvicorn app:app --host 0.0.0.0 --port 8000 --reload

Try it
- Search (gateway forwards to searchd):
  curl "http://localhost:8080/api/search?q=json"

Docker (recommended local run)

Build and run all services:

1) docker compose build
2) docker compose up

Open the web UI: http://localhost:3000

Note: `docker-compose.yml` sets `SEARCHD_SEED_DEMO=1`, so `searchd` will start with a demo document. You can immediately search for `json` in the UI.

Seed sample docs:

curl -X POST http://localhost:8090/index -H 'Content-Type: application/json' \
  -d '{"documents":[{"id":"1","repo":"demo/repo","path":"main.py","language":"python","content":"import json\njson.loads(\"{}\")"}]}'

Then search in the UI for: json

Repo Layout

proto/              # gRPC proto files (draft)
services/
  gateway/          # HTTP API frontend
  searchd/          # Search daemon with in-memory index
ml/
  service/          # FastAPI ML stub

Development

- Makefile provides convenience targets:
  make run-searchd
  make run-gateway
  make run-ml

Roadmap (high-level)
- Replace in-memory index with sharded inverted index + vector store
- Add ingest/indexer pipeline and AST analysis
- Wire gateway->services via gRPC using proto contracts
- Implement semantic search with embeddings (FAISS or custom)
- Add web UI (Next.js) with search UX and code previews

License
TBD


