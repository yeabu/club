ARG PADDLE_BASE_IMAGE=paddlepaddle/paddle:2.6.2
FROM ${PADDLE_BASE_IMAGE}

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PADDLEOCR_MODEL_DIR=/data/paddleocr/models \
    PADDLEOCR_TMP_DIR=/tmp/paddleocr-service \
    PADDLEOCR_LANG=ch \
    PADDLEOCR_CPU_THREADS=4 \
    PADDLEOCR_DISABLE_ONEDNN=true \
    FLAGS_use_mkldnn=0 \
    FLAGS_enable_pir_api=0 \
    FLAGS_enable_pir_in_executor=0

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      curl \
      libglib2.0-0 \
      libgl1 \
      libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

# The base image already contains a PaddlePaddle CPU runtime. Do not install
# paddlepaddle here; mixing a Python slim image and Paddle wheel has caused
# import-time segmentation faults on some x86 hosts.
RUN python -m pip install --upgrade pip setuptools wheel \
    && python -m pip install -r requirements.txt

COPY app.py .

RUN mkdir -p /data/paddleocr/models /tmp/paddleocr-service

EXPOSE 8100

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8100/health || exit 1

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8100"]
