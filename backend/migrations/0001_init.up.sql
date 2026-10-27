CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE courses (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE lectures (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL,
    course_id       TEXT REFERENCES courses(id) ON DELETE SET NULL,
    status          TEXT NOT NULL DEFAULT 'queued',
    progress        REAL NOT NULL DEFAULT 0,
    error           TEXT,
    summary_status  TEXT NOT NULL DEFAULT 'pending',
    source          TEXT NOT NULL,
    source_url      TEXT,
    playback_type   TEXT,
    playback_url    TEXT,
    provider_id     TEXT,
    duration        DOUBLE PRECISION,
    language        TEXT,
    video_path      TEXT,
    audio_path      TEXT,
    thumbnail_path  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE segments (
    lecture_id  TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    idx         INTEGER NOT NULL,
    start_sec   DOUBLE PRECISION NOT NULL,
    end_sec     DOUBLE PRECISION NOT NULL,
    text        TEXT NOT NULL,
    PRIMARY KEY (lecture_id, idx)
);

CREATE TABLE chunks (
    id          BIGSERIAL PRIMARY KEY,
    lecture_id  TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    start_sec   DOUBLE PRECISION NOT NULL,
    end_sec     DOUBLE PRECISION NOT NULL,
    text        TEXT NOT NULL,
    embedding   vector(768) NOT NULL
);

CREATE INDEX chunks_embedding_idx ON chunks
    USING hnsw (embedding vector_cosine_ops);

CREATE TABLE summaries (
    lecture_id  TEXT PRIMARY KEY REFERENCES lectures(id) ON DELETE CASCADE,
    summary     TEXT NOT NULL,
    outline     JSONB NOT NULL,
    key_terms   JSONB NOT NULL
);

CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_idx ON refresh_tokens(user_id);

CREATE TABLE bookmarks (
    id          TEXT PRIMARY KEY,
    client_id   TEXT NOT NULL,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE,
    lecture_id  TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    t           DOUBLE PRECISION NOT NULL,
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX bookmarks_client_lecture_idx ON bookmarks (client_id, lecture_id, t);
CREATE INDEX bookmarks_user_idx ON bookmarks(user_id);

CREATE TABLE watch_progress (
    client_id  TEXT NOT NULL,
    user_id    TEXT REFERENCES users(id) ON DELETE CASCADE,
    lecture_id TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    position   DOUBLE PRECISION NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (client_id, lecture_id)
);

CREATE INDEX watch_progress_client_updated_idx ON watch_progress (client_id, updated_at DESC);
CREATE INDEX watch_progress_user_idx ON watch_progress(user_id);