import os
import tempfile
from fastapi import FastAPI, File, Form, UploadFile, HTTPException
from pydantic import BaseModel
import asr, embed, llm
import logging
import threading
_llm_lock = threading.Lock()

logging.basicConfig(level=logging.INFO)

app = FastAPI(title="ml-service")

@app.get("/health")
def health():
    return {
        "asr": f"faster-whisper {asr.ASR_MODEL}",
        "embeddings": "intfloat/multilingual-e5-base",
        "llm": llm.LLM_FILE.removesuffix(".gguf"),
    }

@app.post("/asr")
async def post_asr(
    audio: UploadFile = File(...),
    language: str | None = Form(None),
):
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        tmp.write(await audio.read())
        path = tmp.name
    try:
        return asr.transcribe(path, language)
    finally:
        os.unlink(path)

class EmbedIn(BaseModel):
    texts: list[str]

@app.post("/embed")
def post_embed(body: EmbedIn):
    return {"vectors": embed.embed(body.texts)}

class EmbedQueryIn(BaseModel):
    text: str


class SummaryIn(BaseModel):
    segments: list[dict]

@app.post("/summary")
def post_summary(body: SummaryIn):
    if not body.segments:
        raise HTTPException(400, "segments empty")
    with _llm_lock:
        return llm.summarize(body.segments)

class Fragment(BaseModel):
    lecture_title: str
    start: float
    end: float
    text: str

class AskIn(BaseModel):
    question: str
    fragments: list[Fragment]
    history: list[dict] = []

@app.post("/ask")
def post_ask(body: AskIn):
    with _llm_lock:
        return {"answer": llm.ask(body.question, [f.model_dump() for f in body.fragments], body.history)}