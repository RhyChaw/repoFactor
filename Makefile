SHELL := /bin/zsh

SEARCHD_URL ?= http://localhost:8090

.PHONY: run-searchd run-gateway run-ml run-devserver proto

run-searchd:
	cd services/searchd && go run ./...

run-gateway:
	cd services/gateway && env SEARCHD_URL=$(SEARCHD_URL) go run ./...

run-ml:
	cd ml/service && python -m venv .venv && source .venv/bin/activate && pip install -r requirements.txt && uvicorn app:app --host 0.0.0.0 --port 8000 --reload

run-devserver:
	cd cmd/devserver && go run ./...

proto:
	@echo "Proto generation to be added (buf/protoc)."


