FROM python:3.12-slim

WORKDIR /app

RUN echo 'Acquire::Retries "10"; Acquire::http::Timeout "120";' > /etc/apt/apt.conf.d/99retries \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        curl \
        ca-certificates \
        build-essential \
        musl \
        libglib2.0-0 \
        libsndfile1 \
        ffmpeg \
    && ln -s /lib/x86_64-linux-musl/libc.so /lib/libc.musl-x86_64.so.1 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

RUN pip install --no-cache-dir \
    --default-timeout=1000 \
    --retries 10 \
    torch --index-url https://download.pytorch.org/whl/cpu

RUN pip install --no-cache-dir \
    --default-timeout=1000 \
    --retries 10 \
    -r requirements.txt

ENV CMAKE_ARGS="-DGGML_NATIVE=ON"
RUN pip install --no-cache-dir \
    --default-timeout=1000 \
    --retries 10 \
    llama-cpp-python==0.3.1

COPY . .

EXPOSE 8001
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8001"]