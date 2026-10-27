import json
import logging
import os
import re
from pathlib import Path
from llama_cpp import Llama

logger = logging.getLogger("llm")

_llm = None
PROMPTS = Path(__file__).parent / "prompts"
MODELS = Path(__file__).parent / "models"
LLM_REPO = os.environ.get("LLM_REPO", "Qwen/Qwen2.5-3B-Instruct-GGUF")
LLM_FILE = os.environ.get("LLM_FILE", "qwen2.5-3b-instruct-q4_k_m.gguf")

SECTION_SEC = 300
MAX_SECTIONS = 12
SECTION_CHARS = 9000
MAX_TERMS = 15

def _get_llm():
    global _llm
    if _llm is None:
        path = MODELS / LLM_FILE
        if not path.exists():
            from huggingface_hub import hf_hub_download
            logger.info("downloading %s/%s", LLM_REPO, LLM_FILE)
            hf_hub_download(LLM_REPO, LLM_FILE, local_dir=MODELS)
        _llm = Llama(
            model_path=str(path),
            n_ctx=8192,
            n_threads=os.cpu_count() or 8,
            verbose=False,
        )
    return _llm

def _extract_json(text: str) -> dict:
    m = re.search(r"\{.*\}", text, re.DOTALL)
    if not m:
        raise ValueError(f"no JSON in LLM output: {text[:200]}")
    return json.loads(m.group(0))

def _clean(text: str) -> str:
    """Small Qwen models leak CJK characters and leave empty list items."""
    text = re.sub(r"[\u3040-\u30ff\u3400-\u9fff\uf900-\ufaff\uff00-\uffef]+", " ", text)
    text = re.sub(r"(?m)^\s*(?:[-*+]|\d+\.)\s*$\n?", "", text)
    return re.sub(r"[ \t]{2,}", " ", text).strip()

def _split_sections(segments: list[dict]) -> list[list[dict]]:
    duration = segments[-1]["end"] - segments[0]["start"]
    n = max(1, min(MAX_SECTIONS, round(duration / SECTION_SEC)))
    n = max(n, min(3, int(duration // 60)))
    target = duration / n
    sections, cur = [], []
    for s in segments:
        if cur and len(sections) < n - 1 and s["start"] - cur[0]["start"] >= target:
            sections.append(cur)
            cur = []
        cur.append(s)
    sections.append(cur)
    return sections

def _section_text(segs: list[dict]) -> str:
    lines = [f"[{s['start']:.0f}] {s['text']}" for s in segs]
    total = sum(len(l) + 1 for l in lines)
    if total > SECTION_CHARS:
        step = total / SECTION_CHARS
        lines = [lines[int(i * step)] for i in range(int(len(lines) / step))]
    return "\n".join(lines)

def _stems(phrase: str) -> list[str]:
    out = []
    for w in re.findall(r"\w+", phrase.lower()):
        if len(w) < 3:
            continue
        out.append(w[: max(3, len(w) - 2)])
    return out

def _occurrences(term: str, segments: list[dict]) -> list[float]:
    stems = _stems(term)
    if not stems:
        return []
    out = []
    for s in segments:
        words = re.findall(r"\w+", s["text"].lower())
        if all(any(w.startswith(st) for w in words) for st in stems):
            out.append(round(float(s["start"]), 2))
    return out

def _summarize_section(llm, idx: int, segs: list[dict]) -> dict:
    prompt = (
        (PROMPTS / "section.txt").read_text()
        .replace("{start}", f"{segs[0]['start']:.0f}")
        .replace("{end}", f"{segs[-1]['end']:.0f}")
        .replace("{segments}", _section_text(segs))
    )
    try:
        resp = llm.create_chat_completion(
            messages=[{"role": "user", "content": prompt}],
            temperature=0.2,
            max_tokens=800,
            response_format={"type": "json_object"},
        )
        return _extract_json(_clean(resp["choices"][0]["message"]["content"]))
    except Exception as e:
        logger.warning("section %d: %s", idx, e)
        return {}

def _children(sub: list, start: float, end: float) -> list[dict]:
    points = []
    for c in sub if isinstance(sub, list) else []:
        if not isinstance(c, dict) or not str(c.get("title", "")).strip():
            continue
        try:
            t = float(c.get("start"))
        except (TypeError, ValueError):
            continue
        if start <= t < end and all(abs(t - p[0]) >= 20 for p in points):
            points.append((t, str(c["title"]).strip()))
    points.sort()
    if len(points) < 2:
        return []
    points[0] = (start, points[0][1])
    return [
        {"title": title, "start": t, "end": points[i + 1][0] if i + 1 < len(points) else end, "summary": None, "children": []}
        for i, (t, title) in enumerate(points)
    ]

def summarize(segments: list[dict]) -> dict:
    llm = _get_llm()
    sections = _split_sections(segments)
    outline, terms = [], {}
    for i, segs in enumerate(sections):
        data = _summarize_section(llm, i, segs)
        start = float(segs[0]["start"]) if i else 0.0
        end = float(sections[i + 1][0]["start"]) if i + 1 < len(sections) else float(segs[-1]["end"])
        summary = str(data.get("summary") or "").strip() or None
        outline.append({
            "title": str(data.get("title") or "").strip() or f"Часть {i + 1}",
            "start": start,
            "end": end,
            "summary": summary,
            "children": _children(data.get("subsections"), start, end),
        })
        for t in data.get("key_terms") or []:
            if not isinstance(t, dict):
                continue
            term = str(t.get("term") or "").strip()
            key = term.lower()
            if term and key not in terms:
                definition = str(t.get("definition") or "").strip() or None
                terms[key] = {"term": term, "definition": definition}

    key_terms = []
    for t in terms.values():
        occ = _occurrences(t["term"], segments)
        if occ:
            key_terms.append({**t, "occurrences": occ})
    key_terms.sort(key=lambda t: -len(t["occurrences"]))
    key_terms = sorted(key_terms[:MAX_TERMS], key=lambda t: t["occurrences"][0])

    plan = "\n".join(
        f"{i + 1}. {o['title']}: {o['summary'] or ''}" for i, o in enumerate(outline)
    )
    prompt = (PROMPTS / "summary.txt").read_text().replace("{sections}", plan)
    resp = llm.create_chat_completion(
        messages=[{"role": "user", "content": prompt}],
        temperature=0.3,
        max_tokens=1200,
    )
    summary = _clean(resp["choices"][0]["message"]["content"])
    summary = re.sub(r"^```(?:markdown)?\s*|\s*```$", "", summary)
    summary = re.sub(r"(?m)^#\s+", "## ", summary)
    return {"summary": summary, "outline": outline, "key_terms": key_terms}

def ask(question: str, fragments: list[dict], history: list[dict]) -> str:
    llm = _get_llm()
    frags_text = "\n".join(
        f"[{i+1}] ({f['lecture_title']}, {f['start']:.1f}-{f['end']:.1f}) {f['text']}"
        for i, f in enumerate(fragments)
    )
    hist_text = "\n".join(f"{h['role']}: {h['content']}" for h in history[-10:]) or "(пусто)"
    prompt = (
        (PROMPTS / "ask.txt").read_text()
        .replace("{fragments}", frags_text)
        .replace("{history}", hist_text)
        .replace("{question}", question)
    )
    resp = llm.create_chat_completion(
        messages=[{"role": "user", "content": prompt}],
        temperature=0.2,
        max_tokens=1024,
    )
    return _clean(resp["choices"][0]["message"]["content"])
