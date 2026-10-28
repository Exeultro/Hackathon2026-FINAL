# Lecture Search — семантический поиск по видеолекциям

## Стек

- **Backend:** Go 1.26, chi, pgx/v5
- **ML:** Python 3.12, FastAPI, faster-whisper (large-v3-turbo), multilingual-e5-base, llama.cpp (Qwen2.5-3B Q4_K_M)
- **БД:** PostgreSQL 16 + pgvector
- **Пайплайн:** ffmpeg → Whisper → e5 → pgvector
- **RAG:** поиск по чанкам → LLM → ответ со ссылками

## Быстрый старт

### 1. Предварительно

- Docker Desktop с WSL2
- ~15 ГБ свободного места (образ ML ~5 ГБ, модели ~2 ГБ)

### 2. Запуск

```bash
docker compose up --build
```

API на http://localhost:8000/api/v1. Модели скачиваются при первом обращении: Whisper и e5 из Hugging Face, GGUF Qwen2.5-3B в `ml-service/models/`. Если файл модели уже лежит там, он используется как есть.

Переменные: `PUBLIC_URL` - адрес API, который видит браузер (из него строятся ссылки на видео и обложки, по умолчанию `http://localhost:8000`); `ASR_MODEL` у ML-сервиса (по умолчанию `large-v3-turbo`).
