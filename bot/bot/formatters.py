import html
import re


def fmt_timecode(sec: float) -> str:
    sec = int(sec)
    h, rem = divmod(sec, 3600)
    m, s = divmod(rem, 60)
    if h:
        return f"{h}:{m:02d}:{s:02d}"
    return f"{m}:{s:02d}"


def front_link(front_url: str, lecture_id: str, sec: float) -> str:
    return f"{front_url}/lectures/{lecture_id}?t={int(sec)}"


def youtube_link(provider_id: str, sec: float) -> str:
    return f"https://youtu.be/{provider_id}?t={int(sec)}"


def youtube_id(url: str | None) -> str | None:
    m = re.search(r"(?:youtu\.be/|[?&]v=|/shorts/|/embed/|/live/)([\w-]{11})", url or "")
    return m.group(1) if m else None


def playbacks_of(lectures: list) -> dict:
    """lecture id -> playback for the YouTube link: the backend always serves the downloaded file."""
    out = {}
    for lec in lectures:
        pb = lec.get("playback") or {}
        if pb.get("type") == "youtube" and pb.get("provider_id"):
            out[lec["id"]] = pb
        elif lec.get("source") == "youtube" and youtube_id(lec.get("source_url")):
            out[lec["id"]] = {"type": "youtube", "provider_id": youtube_id(lec.get("source_url"))}
    return out


def _u16_to_py(s: str, i: int) -> int:
    """UTF-16 offset (as in the API) -> Python str index."""
    return len(s.encode("utf-16-le")[: i * 2].decode("utf-16-le", "ignore"))


def _highlight_escape(text: str, spans: list[tuple[int, int]]) -> str:
    """Экранирует текст и оборачивает спаны (индексы Python) в <b>."""
    out = []
    pos = 0
    for start, end in sorted(spans):
        if start < pos:
            continue
        out.append(html.escape(text[pos:start]))
        out.append("<b>" + html.escape(text[start:end]) + "</b>")
        pos = end
    out.append(html.escape(text[pos:]))
    return "".join(out)


def _clip_around(text: str, spans: list[tuple[int, int]], around: int = 200) -> tuple[str, list[tuple[int, int]]]:
    """Обрезает текст до ~around символов вокруг первого спана (индексы Python)."""
    if len(text) <= around:
        return text, spans
    first = spans[0][0] if spans else 0
    lo = max(0, first - around // 3)
    hi = min(len(text), lo + around)
    while 0 < lo < first and not text[lo - 1].isspace():
        lo += 1
    while hi < len(text) and not text[hi].isspace():
        hi -= 1
        if hi <= lo:
            hi = min(len(text), lo + around)
            break
    prefix = "…" if lo > 0 else ""
    suffix = "…" if hi < len(text) else ""
    shift = len(prefix) - lo
    new_spans = [(s + shift, e + shift) for s, e in spans if s >= lo and e <= hi]
    return prefix + text[lo:hi] + suffix, new_spans


def format_hit(idx: int, hit: dict, front_url: str, playback: dict | None = None) -> str:
    lec = hit.get("lecture") or {}
    sec = hit.get("start", 0)
    link = front_link(front_url, lec.get("id", ""), sec)
    text = hit.get("text", "")
    spans = [(_u16_to_py(text, s), _u16_to_py(text, e)) for s, e in hit.get("highlights") or []]
    clipped, spans = _clip_around(text, spans)
    body = _highlight_escape(clipped, spans)
    line = f'{idx}. {html.escape(lec.get("title", "Лекция"))} — <a href="{link}">{fmt_timecode(sec)}</a>\n   «{body}»'
    if playback and playback.get("type") == "youtube" and playback.get("provider_id"):
        line += f'\n   <a href="{youtube_link(playback["provider_id"], sec)}">на YouTube</a>'
    return line


def format_search_results(data: dict, front_url: str, shown: int = 5,
                          playbacks: dict | None = None) -> str:
    hits = data.get("hits") or []
    if not hits:
        return "Ничего не нашлось. Попробуйте сформулировать иначе."
    header = f"Нашлось {len(hits)} фрагментов"
    if data.get("took_ms") is not None:
        header += f" · {data['took_ms']} мс"
    lines = [header, ""]
    for idx, hit in enumerate(hits[:shown], start=1):
        pb = (playbacks or {}).get((hit.get("lecture") or {}).get("id"))
        lines.append(format_hit(idx, hit, front_url, pb))
    return "\n".join(lines)


def md_to_html(md: str) -> str:
    """Минимальный Markdown -> Telegram HTML: заголовки и **жирный** в <b>, списки маркерами."""
    out = []
    for line in md.splitlines():
        m = re.match(r"^\s*#{1,6}\s+(.*)$", line)
        if m:
            out.append(f"<b>{_inline(m.group(1))}</b>")
            continue
        m = re.match(r"^(\s*)[-*+]\s+(.*)$", line)
        if m:
            out.append(f"{m.group(1)}• {_inline(m.group(2))}")
            continue
        out.append(_inline(line))
    return "\n".join(out).strip()


def _inline(s: str) -> str:
    s = html.escape(s)
    s = re.sub(r"\*\*(.+?)\*\*", r"<b>\1</b>", s)
    s = re.sub(r"(?<![\w*])\*(?!\s)(.+?)(?<!\s)\*(?![\w*])", r"<i>\1</i>", s)
    s = re.sub(r"`([^`]+)`", r"<code>\1</code>", s)
    return s


def format_ask_answer(data: dict, front_url: str) -> str:
    sources = data.get("sources") or []
    answer = md_to_html(data.get("answer", ""))

    def ref(m: re.Match) -> str:
        i = int(m.group(1))
        if 1 <= i <= len(sources):
            s = sources[i - 1]
            url = front_link(front_url, (s.get("lecture") or {}).get("id", ""), s.get("start", 0))
            return f'<a href="{url}">[{i}]</a>'
        return m.group(0)

    answer = re.sub(r"\[(\d+)\]", ref, answer)
    parts = [answer or "Ответа нет."]
    if sources:
        parts.append("\n<b>Источники:</b>")
        for i, s in enumerate(sources, start=1):
            lec = s.get("lecture") or {}
            url = front_link(front_url, lec.get("id", ""), s.get("start", 0))
            parts.append(f'{i}. {html.escape(lec.get("title", "Лекция"))} — <a href="{url}">{fmt_timecode(s.get("start", 0))}</a>')
    return "\n".join(parts)
