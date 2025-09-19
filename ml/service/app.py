from fastapi import FastAPI
from pydantic import BaseModel
from typing import List

app = FastAPI(title="PolyScale ML Service", version="0.0.1")


class EmbedRequest(BaseModel):
    texts: List[str]


class EmbedResponse(BaseModel):
    vectors: List[List[float]]


class RefactorRequest(BaseModel):
    code: str
    instruction: str


class RefactorResponse(BaseModel):
    patch: str


@app.get("/healthz")
def healthz():
    return {"status": "ok"}


@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest):
    vectors = [[float(len(t))] for t in req.texts]
    return {"vectors": vectors}


@app.post("/refactor", response_model=RefactorResponse)
def refactor(req: RefactorRequest):
    patch = "noop"
    return {"patch": patch}


