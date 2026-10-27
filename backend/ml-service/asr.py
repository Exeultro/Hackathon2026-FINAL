import os
from faster_whisper import WhisperModel
import traceback, logging
logger = logging.getLogger("asr")

ASR_MODEL = os.environ.get("ASR_MODEL", "large-v3-turbo")

_model = None

def _get_model():
    global _model
    if _model is None:
        _model = WhisperModel(ASR_MODEL, device="cpu", compute_type="int8", cpu_threads=os.cpu_count() or 4)
    return _model

def transcribe(audio_path: str, language: str | None = None) -> dict:
    try:
        model = _get_model()
        segments_iter, info = model.transcribe(
            audio_path,
            language=None if language in (None, "auto") else language,
            vad_filter=True,
        )
        segments = []
        for s in segments_iter:
            text = s.text.strip()
            if not text:
                continue
            segments.append({"start": float(s.start), "end": float(s.end), "text": text})
        return {
            "language": info.language or "unknown",
            "duration": float(info.duration),
            "segments": segments,
        }
    except Exception as e:
        logger.error("ASR failed: %s", e)
        logger.error(traceback.format_exc())
        raise
