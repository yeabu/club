FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    AI_WORKER_PORT=8090 \
    AI_WORKER_RESULT_DIR=/data/ai-worker/results \
    AI_WORKER_SAMPLE_MANIFEST=/app/samples/manifest.json

WORKDIR /app

COPY app ./app
COPY samples ./samples

RUN mkdir -p /data/ai-worker/results

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8090/health', timeout=3).read()"

CMD ["python", "-m", "app.main"]
