from __future__ import annotations

import base64
import logging
import os
import tempfile
import time
from pathlib import Path
from typing import Any

from fastapi import FastAPI, File, HTTPException, UploadFile
from pydantic import BaseModel


app = FastAPI(title="Club PaddleOCR CPU Service", version="0.1.0")
logger = logging.getLogger("paddleocr-service")

_ocr: Any | None = None
_model_loaded_at: float | None = None


class OCRBase64Request(BaseModel):
    fileName: str = "scan.jpg"
    contentBase64: str


def env_bool(name: str, default: bool) -> bool:
    raw = os.getenv(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


def env_int(name: str, default: int) -> int:
    raw = os.getenv(name)
    if not raw:
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def service_config() -> dict[str, Any]:
    return {
        "lang": os.getenv("PADDLEOCR_LANG", "ch"),
        "useAngleCls": env_bool("PADDLEOCR_USE_ANGLE_CLS", True),
        "cpuThreads": env_int("PADDLEOCR_CPU_THREADS", 4),
        "disableOneDNN": env_bool("PADDLEOCR_DISABLE_ONEDNN", True),
        "modelDir": os.getenv("PADDLEOCR_MODEL_DIR", "/data/paddleocr/models"),
        "tmpDir": os.getenv("PADDLEOCR_TMP_DIR", "/tmp/paddleocr-service"),
        "maxUploadMb": env_int("PADDLEOCR_MAX_UPLOAD_MB", 25),
    }


def get_ocr() -> Any:
    global _ocr, _model_loaded_at
    if _ocr is not None:
        return _ocr

    config = service_config()
    Path(config["modelDir"]).mkdir(parents=True, exist_ok=True)
    os.environ.setdefault("PADDLE_HOME", config["modelDir"])
    if config["disableOneDNN"]:
        # PaddleOCR 3.x + CPU PaddlePaddle can hit oneDNN/PIR runtime gaps on
        # some x86 hosts. Prefer the plain CPU executor for deployment safety.
        os.environ.setdefault("FLAGS_use_mkldnn", "0")
        os.environ.setdefault("FLAGS_enable_pir_api", "0")
        os.environ.setdefault("FLAGS_enable_pir_in_executor", "0")

    from paddleocr import PaddleOCR

    try:
        # PaddleOCR 3.x rejects several 2.x constructor arguments such as
        # use_gpu/cpu_threads. CPU is already controlled by installing the CPU
        # paddlepaddle wheel, so the minimal constructor is the safest default.
        _ocr = PaddleOCR(lang=config["lang"])
    except Exception as exc:
        logger.warning("PaddleOCR minimal constructor failed, retrying legacy constructor: %s", exc)
        _ocr = PaddleOCR(
            lang=config["lang"],
            use_gpu=False,
            show_log=False,
            use_angle_cls=config["useAngleCls"],
            cpu_threads=config["cpuThreads"],
        )
    _model_loaded_at = time.time()
    return _ocr


def check_upload_size(content: bytes) -> None:
    max_bytes = service_config()["maxUploadMb"] * 1024 * 1024
    if len(content) == 0:
        raise HTTPException(status_code=400, detail="file is empty")
    if len(content) > max_bytes:
        raise HTTPException(status_code=413, detail=f"file exceeds {max_bytes} bytes")


def safe_suffix(file_name: str) -> str:
    suffix = Path(file_name).suffix.lower()
    if suffix in {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}:
        return suffix
    return ".jpg"


def write_temp_file(file_name: str, content: bytes) -> Path:
    check_upload_size(content)
    tmp_dir = Path(service_config()["tmpDir"])
    tmp_dir.mkdir(parents=True, exist_ok=True)
    handle = tempfile.NamedTemporaryFile(delete=False, suffix=safe_suffix(file_name), dir=tmp_dir)
    try:
        handle.write(content)
        return Path(handle.name)
    finally:
        handle.close()


def run_paddleocr(path: Path) -> Any:
    ocr = get_ocr()
    if hasattr(ocr, "predict"):
        return list(ocr.predict(str(path)))
    if hasattr(ocr, "ocr"):
        try:
            return ocr.ocr(str(path), cls=service_config()["useAngleCls"])
        except TypeError:
            return ocr.ocr(str(path))
    raise HTTPException(status_code=500, detail="unsupported PaddleOCR instance")


def normalize_ocr_result(raw: Any) -> dict[str, Any]:
    blocks: list[dict[str, Any]] = []

    def add_block(text: Any, confidence: Any = None, box: Any = None) -> None:
        value = str(text or "").strip()
        if not value:
            return
        try:
            score = float(confidence) if confidence is not None else None
        except (TypeError, ValueError):
            score = None
        blocks.append({"text": value, "confidence": score, "box": to_jsonable(box)})

    if isinstance(raw, list):
        for page in raw:
            if isinstance(page, dict):
                parse_result_dict(page, add_block)
                continue
            if not isinstance(page, list):
                continue
            for item in page:
                if isinstance(item, dict):
                    parse_result_dict(item, add_block)
                    continue
                if isinstance(item, (list, tuple)) and len(item) >= 2:
                    box = item[0]
                    rec = item[1]
                    if isinstance(rec, (list, tuple)) and len(rec) >= 2:
                        add_block(rec[0], rec[1], box)
                    elif isinstance(rec, str):
                        add_block(rec, None, box)
    elif isinstance(raw, dict):
        parse_result_dict(raw, add_block)

    return {
        "provider": "paddleocr",
        "text": "\n".join(block["text"] for block in blocks),
        "blocks": blocks,
    }


def parse_result_dict(item: dict[str, Any], add_block: Any) -> None:
    texts = item.get("rec_texts") or item.get("texts")
    scores = item.get("rec_scores") or item.get("scores") or []
    boxes = item.get("rec_polys") or item.get("dt_polys") or item.get("boxes") or []
    if isinstance(texts, list):
        for index, text in enumerate(texts):
            score = scores[index] if isinstance(scores, list) and index < len(scores) else None
            box = boxes[index] if isinstance(boxes, list) and index < len(boxes) else None
            add_block(text, score, box)
        return
    add_block(item.get("text") or item.get("rec_text"), item.get("confidence") or item.get("score"), item.get("box"))


def to_jsonable(value: Any) -> Any:
    if value is None or isinstance(value, (str, int, float, bool)):
        return value
    if hasattr(value, "tolist"):
        return to_jsonable(value.tolist())
    if isinstance(value, dict):
        return {str(key): to_jsonable(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [to_jsonable(item) for item in value]
    return str(value)


def ocr_bytes(file_name: str, content: bytes) -> dict[str, Any]:
    path = write_temp_file(file_name, content)
    started = time.perf_counter()
    try:
        try:
            raw = run_paddleocr(path)
        except Exception as exc:
            logger.exception("PaddleOCR recognition failed")
            raise HTTPException(status_code=502, detail=f"PaddleOCR recognition failed: {exc}") from exc
        result = normalize_ocr_result(raw)
        result["elapsedMs"] = int((time.perf_counter() - started) * 1000)
        result["fileName"] = file_name
        return result
    finally:
        try:
            path.unlink(missing_ok=True)
        except OSError:
            pass


@app.get("/health")
def health() -> dict[str, Any]:
    return {
        "status": "ok",
        "modelLoaded": _ocr is not None,
        "modelLoadedAt": _model_loaded_at,
        "config": service_config(),
    }


@app.post("/warmup")
def warmup() -> dict[str, Any]:
    started = time.perf_counter()
    get_ocr()
    return {"status": "ok", "elapsedMs": int((time.perf_counter() - started) * 1000)}


@app.post("/ocr")
async def ocr_file(file: UploadFile = File(...)) -> dict[str, Any]:
    content = await file.read()
    return ocr_bytes(file.filename or "scan.jpg", content)


@app.post("/ocr/base64")
def ocr_base64(payload: OCRBase64Request) -> dict[str, Any]:
    try:
        content = base64.b64decode(payload.contentBase64, validate=True)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail="invalid base64 content") from exc
    return ocr_bytes(payload.fileName, content)
