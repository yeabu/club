from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
import socket
import ssl
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any


def load_dotenv(name: str = ".env.local") -> None:
    current = Path.cwd()
    for directory in [current, *current.parents]:
        candidate = directory / name
        if not candidate.exists():
            continue
        for raw_line in candidate.read_text(encoding="utf-8").splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            key = key.strip()
            value = value.strip().strip("\"'")
            if key and key not in os.environ:
                os.environ[key] = value
        return


def env_bool(name: str, default: bool = False) -> bool:
    raw = os.getenv(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


@dataclass
class WorkerConfig:
    app_env: str
    port: int
    redis_addr: str
    redis_db: int
    redis_password: str
    redis_stream: str
    redis_group: str
    redis_consumer: str
    api_base_url: str
    paddleocr_service_url: str
    result_dir: Path
    sample_manifest: Path
    model_version: str
    storage_driver: str
    enable_redis_consumer: bool
    ai_provider: str
    anthropic_base_url: str
    anthropic_auth_token: str
    anthropic_model: str
    ai_timeout_seconds: int

    @classmethod
    def from_env(cls) -> "WorkerConfig":
        anthropic_model = os.getenv("ANTHROPIC_MODEL", "").strip()
        return cls(
            app_env=os.getenv("APP_ENV", "development"),
            port=int(os.getenv("AI_WORKER_PORT", "8090")),
            redis_addr=os.getenv("REDIS_ADDR", "127.0.0.1:6379"),
            redis_db=int(os.getenv("REDIS_DB", "0")),
            redis_password=os.getenv("REDIS_PASSWORD", ""),
            redis_stream=os.getenv("AI_WORKER_SCAN_STREAM", "club:scan:tasks"),
            redis_group=os.getenv("AI_WORKER_REDIS_GROUP", "club-ai-worker"),
            redis_consumer=os.getenv("AI_WORKER_REDIS_CONSUMER", socket.gethostname()),
            api_base_url=os.getenv("API_BASE_URL", "http://localhost:8080").rstrip("/"),
            paddleocr_service_url=os.getenv("PADDLEOCR_SERVICE_URL", "").strip().rstrip("/"),
            result_dir=Path(os.getenv("AI_WORKER_RESULT_DIR", "var/ai-worker/results")),
            sample_manifest=Path(os.getenv("AI_WORKER_SAMPLE_MANIFEST", "samples/manifest.json")),
            model_version=os.getenv("AI_MODEL_VERSION", anthropic_model or "mock-ai-worker-v1"),
            storage_driver=os.getenv("STORAGE_DRIVER", "minio"),
            enable_redis_consumer=env_bool("AI_WORKER_CONSUME_REDIS", False),
            ai_provider=os.getenv("AI_PROVIDER", "").strip().lower(),
            anthropic_base_url=os.getenv("ANTHROPIC_BASE_URL", "").strip().rstrip("/"),
            anthropic_auth_token=os.getenv("ANTHROPIC_AUTH_TOKEN", "").strip(),
            anthropic_model=anthropic_model,
            ai_timeout_seconds=int(os.getenv("AI_TIMEOUT_SECONDS", "45")),
        )

    def public(self) -> dict[str, Any]:
        return {
            "appEnv": self.app_env,
            "port": self.port,
            "storageDriver": self.storage_driver,
            "modelVersion": self.model_version,
            "redis": {
                "addr": self.redis_addr,
                "db": self.redis_db,
                "stream": self.redis_stream,
                "group": self.redis_group,
                "passwordProvided": bool(self.redis_password),
                "consumerEnabled": self.enable_redis_consumer,
            },
            "api": {
                "baseUrl": self.api_base_url,
            },
            "ocr": {
                "provider": "remote-paddleocr" if self.paddleocr_service_url else ("paddleocr" if env_bool("AI_WORKER_USE_PADDLEOCR", False) else "mock-ocr"),
                "paddleocrServiceUrl": self.paddleocr_service_url,
            },
            "ai": {
                "provider": self.ai_provider or ("anthropic" if self.anthropic_auth_token else "mock"),
                "baseUrl": self.anthropic_base_url,
                "model": self.anthropic_model or self.model_version,
                "authTokenConfigured": bool(self.anthropic_auth_token),
                "timeoutSeconds": self.ai_timeout_seconds,
            },
            "samples": {
                "manifest": str(self.sample_manifest),
            },
        }


def log_event(event: str, **fields: Any) -> None:
    payload = {
        "ts": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "event": event,
        **fields,
    }
    print(json.dumps(payload, ensure_ascii=False), flush=True)


def elapsed_ms(start: float) -> int:
    return int((time.perf_counter() - start) * 1000)


class MockOCRAdapter:
    name = "mock-ocr"

    def extract(self, file_ref: dict[str, Any]) -> dict[str, Any]:
        key = str(file_ref.get("key") or file_ref.get("url") or "unknown")
        confidence = 82 + stable_number(key, 15)
        return {
            "provider": self.name,
            "objectKey": key,
            "text": file_ref.get("mockText")
            or "设需要 x 千克，3/5 = x/40，5x = 120，x = 24。答需要 24 千克。",
            "confidence": confidence,
            "blocks": [
                {
                    "text": "设需要 x 千克",
                    "confidence": min(confidence + 2, 99),
                    "box": {"x": 120, "y": 420, "width": 180, "height": 36},
                },
                {
                    "text": "x = 24",
                    "confidence": confidence,
                    "box": {"x": 128, "y": 480, "width": 96, "height": 34},
                },
            ],
        }


class PaddleOCRAdapter(MockOCRAdapter):
    name = "paddleocr"

    def __init__(self) -> None:
        if importlib.util.find_spec("paddleocr") is None:
            raise RuntimeError("paddleocr package is not installed")


class RemotePaddleOCRAdapter(MockOCRAdapter):
    name = "remote-paddleocr"

    def __init__(self, base_url: str, api_base_url: str = "") -> None:
        self.base_url = base_url.rstrip("/")
        self.api_base_url = api_base_url.rstrip("/")
        if not self.base_url:
            raise RuntimeError("PADDLEOCR_SERVICE_URL is empty")

    def extract(self, file_ref: dict[str, Any]) -> dict[str, Any]:
        content = self._load_content(file_ref)
        if content is None:
            raise RuntimeError("remote PaddleOCR requires file content or a readable local path")
        file_name = str(file_ref.get("fileName") or file_ref.get("key") or "scan.jpg")
        payload = {
            "fileName": file_name,
            "contentBase64": base64_encode(content),
        }
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        request = urllib.request.Request(
            f"{self.base_url}/ocr/base64",
            data=body,
            method="POST",
            headers={"Content-Type": "application/json; charset=utf-8"},
        )
        with urllib.request.urlopen(request, timeout=30) as response:
            result = json.loads(response.read().decode("utf-8"))
        return {
            "provider": self.name,
            "objectKey": file_ref.get("key") or file_ref.get("url") or file_name,
            "text": result.get("text", ""),
            "confidence": average_confidence(result.get("blocks", [])),
            "blocks": result.get("blocks", []),
            "elapsedMs": result.get("elapsedMs", 0),
        }

    def _load_content(self, file_ref: dict[str, Any]) -> bytes | None:
        raw_base64 = file_ref.get("contentBase64")
        if isinstance(raw_base64, str) and raw_base64:
            import base64

            return base64.b64decode(raw_base64)
        for key in ("localPath", "path"):
            raw_path = file_ref.get(key)
            if isinstance(raw_path, str) and raw_path:
                path = Path(raw_path)
                if path.exists() and path.is_file():
                    return path.read_bytes()
        raw_url = file_ref.get("url")
        if isinstance(raw_url, str) and raw_url.startswith("/") and self.api_base_url:
            raw_url = self.api_base_url + raw_url
        if isinstance(raw_url, str) and raw_url.startswith(("http://", "https://")):
            with urllib.request.urlopen(raw_url, timeout=15) as response:
                return response.read()
        return None


def base64_encode(content: bytes) -> str:
    import base64

    return base64.b64encode(content).decode("ascii")


def average_confidence(blocks: Any) -> int:
    if not isinstance(blocks, list) or not blocks:
        return 0
    scores: list[float] = []
    for block in blocks:
        if not isinstance(block, dict):
            continue
        score = block.get("confidence")
        if isinstance(score, (int, float)):
            scores.append(float(score))
    if not scores:
        return 0
    average = sum(scores) / len(scores)
    if average <= 1:
        average *= 100
    return int(max(0, min(100, average)))


def build_ocr_adapter() -> MockOCRAdapter:
    service_url = os.getenv("PADDLEOCR_SERVICE_URL", "").strip()
    if service_url:
        return RemotePaddleOCRAdapter(service_url, os.getenv("API_BASE_URL", "http://localhost:8080"))
    if env_bool("AI_WORKER_USE_PADDLEOCR", False):
        try:
            return PaddleOCRAdapter()
        except RuntimeError as exc:
            log_event("ocr_adapter_fallback", reason=str(exc), fallback="mock-ocr")
    return MockOCRAdapter()


def preprocess_scan(file_ref: dict[str, Any]) -> dict[str, Any]:
    key = str(file_ref.get("key") or file_ref.get("url") or "unknown")
    skew = stable_number(key, 7) - 3
    return {
        "objectKey": key,
        "rotationApplied": -skew,
        "deskewDegrees": skew,
        "grayscale": True,
        "denoise": True,
        "cropBox": {"x": 0, "y": 0, "width": 2480, "height": 3508},
        "outputKey": f"processed/{key}".replace("//", "/"),
    }


def recognize_omr(file_ref: dict[str, Any], template: dict[str, Any] | None = None) -> dict[str, Any]:
    key = str(file_ref.get("key") or file_ref.get("url") or "unknown")
    answers = []
    question_count = int((template or {}).get("objectiveQuestionCount") or 5)
    choices = ["A", "B", "C", "D"]
    for index in range(1, question_count + 1):
        selected = choices[stable_number(f"{key}:{index}", len(choices))]
        answers.append(
            {
                "questionNo": str(index),
                "selected": selected,
                "confidence": 84 + stable_number(f"{key}:{index}:confidence", 14),
                "box": {"x": 120, "y": 220 + index * 48, "width": 360, "height": 32},
            }
        )
    return {"provider": "mock-omr", "answers": answers}


class AIAdapter:
    name = "mock"

    def analyze_paper(self, payload: dict[str, Any]) -> dict[str, Any]:
        return mock_analyze_paper(payload)

    def grade_subjective(self, payload: dict[str, Any]) -> dict[str, Any]:
        return mock_grade_subjective(payload)

    def detect_wrong_reason(self, payload: dict[str, Any]) -> dict[str, Any]:
        return mock_detect_wrong_reason(payload)


class AnthropicCompatibleAIAdapter(AIAdapter):
    name = "anthropic-compatible"

    def __init__(self, base_url: str, auth_token: str, model: str, timeout_seconds: int = 45) -> None:
        self.base_url = base_url.rstrip("/")
        self.auth_token = auth_token
        self.model = model
        self.timeout_seconds = timeout_seconds
        self.ssl_context = make_ssl_context()
        if not self.base_url:
            raise RuntimeError("ANTHROPIC_BASE_URL is empty")
        if not self.auth_token:
            raise RuntimeError("ANTHROPIC_AUTH_TOKEN is empty")
        if not self.model:
            raise RuntimeError("ANTHROPIC_MODEL is empty")

    def analyze_paper(self, payload: dict[str, Any]) -> dict[str, Any]:
        fallback = mock_analyze_paper(payload)
        prompt = (
            "请根据输入的试卷扫描/OCR上下文生成答题卡模板建议。"
            "只返回 JSON，不要 Markdown。字段必须包含：paperName, questionCount, totalScore,"
            " suggestedQuestions, reviewRequired。suggestedQuestions 每项包含 id,no,questionNo,type,"
            " score,standardAnswer,scoringRules,knowledge,region。region 包含 page,x,y,width,height。"
            "\n输入：\n" + json.dumps(payload, ensure_ascii=False)
        )
        result = self._request_json("paper-analyze", prompt, fallback)
        if result.pop("_aiProviderFallback", False):
            return fallback
        result["paperName"] = str(result.get("paperName") or fallback["paperName"])
        result["questionCount"] = to_int(result.get("questionCount"), fallback["questionCount"])
        result["totalScore"] = to_float(result.get("totalScore"), fallback["totalScore"])
        result["suggestedQuestions"] = normalize_suggested_questions(result.get("suggestedQuestions"), fallback["suggestedQuestions"])
        result["reviewRequired"] = bool(result.get("reviewRequired", True))
        result["source"] = self.name
        result["modelVersion"] = self.model
        return result

    def grade_subjective(self, payload: dict[str, Any]) -> dict[str, Any]:
        fallback = mock_grade_subjective(payload)
        prompt = (
            "你是阅卷系统的主观题评分助手。请严格根据标准答案、评分规则和学生OCR文本给出建议分。"
            "只返回 JSON，不要 Markdown。字段必须包含：score,fullScore,reason,evidence,comments,confidence。"
            "score 不得超过 fullScore，confidence 为 0-100。"
            "\n输入：\n" + json.dumps(payload, ensure_ascii=False)
        )
        result = self._request_json("subjective-grading", prompt, fallback)
        if result.pop("_aiProviderFallback", False):
            return fallback
        full_score = to_float(result.get("fullScore"), to_float(payload.get("fullScore") or payload.get("score"), fallback["fullScore"]))
        score = min(full_score, max(0.0, to_float(result.get("score"), fallback["score"])))
        return {
            "score": round(score, 1),
            "fullScore": full_score,
            "reason": str(result.get("reason") or fallback["reason"]),
            "evidence": normalize_string_list(result.get("evidence"), fallback["evidence"]),
            "comments": normalize_string_list(result.get("comments"), fallback["comments"]),
            "confidence": clamp_int(result.get("confidence"), fallback["confidence"], 0, 100),
            "modelVersion": self.model,
            "source": self.name,
        }

    def detect_wrong_reason(self, payload: dict[str, Any]) -> dict[str, Any]:
        fallback = mock_detect_wrong_reason(payload)
        prompt = (
            "你是学情分析助手。请根据学生答案、标准答案、知识点和评分情况判断错因。"
            "只返回 JSON，不要 Markdown。字段必须包含：wrongReason,errorType,knowledge,trainingHint,confidence。"
            "confidence 为 0-100。"
            "\n输入：\n" + json.dumps(payload, ensure_ascii=False)
        )
        result = self._request_json("wrong-reason", prompt, fallback)
        if result.pop("_aiProviderFallback", False):
            return fallback
        return {
            "wrongReason": str(result.get("wrongReason") or fallback["wrongReason"]),
            "errorType": str(result.get("errorType") or fallback["errorType"]),
            "knowledge": normalize_string_list(result.get("knowledge"), fallback["knowledge"]),
            "trainingHint": str(result.get("trainingHint") or fallback["trainingHint"]),
            "confidence": clamp_int(result.get("confidence"), fallback["confidence"], 0, 100),
            "modelVersion": self.model,
            "source": self.name,
        }

    def _request_json(self, task: str, prompt: str, fallback: dict[str, Any]) -> dict[str, Any]:
        body = json.dumps(
            {
                "model": self.model,
                "max_tokens": 1800,
                "temperature": 0.2,
                "system": "你是阅卷与学情平台的结构化 JSON 任务执行器。必须只输出一个合法 JSON 对象。",
                "messages": [{"role": "user", "content": prompt}],
            },
            ensure_ascii=False,
        ).encode("utf-8")
        request = urllib.request.Request(
            anthropic_messages_url(self.base_url),
            data=body,
            method="POST",
            headers={
                "Content-Type": "application/json; charset=utf-8",
                "x-api-key": self.auth_token,
                "Authorization": f"Bearer {self.auth_token}",
                "anthropic-version": "2023-06-01",
            },
        )
        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds, context=self.ssl_context) as response:
                response_body = response.read().decode("utf-8")
            return parse_ai_json_response(json.loads(response_body))
        except Exception as exc:
            log_event("ai_provider_failed", provider=self.name, task=task, model=self.model, error=str(exc), fallback="mock")
            result = dict(fallback)
            result["_aiProviderFallback"] = True
            return result


def build_ai_adapter(config: WorkerConfig | None = None) -> AIAdapter:
    provider = (config.ai_provider if config else os.getenv("AI_PROVIDER", "")).strip().lower()
    base_url = (config.anthropic_base_url if config else os.getenv("ANTHROPIC_BASE_URL", "")).strip().rstrip("/")
    auth_token = (config.anthropic_auth_token if config else os.getenv("ANTHROPIC_AUTH_TOKEN", "")).strip()
    model = (config.anthropic_model if config else os.getenv("ANTHROPIC_MODEL", "")).strip()
    timeout_seconds = config.ai_timeout_seconds if config else int(os.getenv("AI_TIMEOUT_SECONDS", "45"))
    if provider in {"", "anthropic", "anthropic-compatible", "deepseek"} and auth_token:
        return AnthropicCompatibleAIAdapter(base_url, auth_token, model, timeout_seconds)
    return AIAdapter()


def analyze_paper(payload: dict[str, Any]) -> dict[str, Any]:
    return build_ai_adapter().analyze_paper(payload)


def grade_subjective(payload: dict[str, Any]) -> dict[str, Any]:
    return build_ai_adapter().grade_subjective(payload)


def detect_wrong_reason(payload: dict[str, Any]) -> dict[str, Any]:
    return build_ai_adapter().detect_wrong_reason(payload)


def mock_analyze_paper(payload: dict[str, Any]) -> dict[str, Any]:
    paper_name = payload.get("paperName") or payload.get("title") or "未命名试卷"
    return {
        "paperName": paper_name,
        "questionCount": 25,
        "totalScore": 100,
        "suggestedQuestions": [
            {
                "id": "ai_q_001",
                "no": "1",
                "questionNo": "1",
                "type": "single_choice",
                "score": 2,
                "standardAnswer": "A",
                "scoringRules": ["选对 A 得 2 分"],
                "knowledge": ["分数"],
                "region": {"page": 1, "x": 120, "y": 260, "width": 480, "height": 80},
            },
            {
                "id": "ai_q_015",
                "no": "15",
                "questionNo": "15",
                "type": "subjective",
                "score": 10,
                "standardAnswer": "先设未知数 x，列出比例关系 3:5 = x:40，解得 x = 24。",
                "scoringRules": ["正确设未知数 2 分", "列出比例关系 4 分", "计算过程正确 2 分", "答语完整 2 分"],
                "knowledge": ["比例", "应用题建模"],
                "region": {"page": 2, "x": 96, "y": 420, "width": 620, "height": 180},
            },
            {
                "id": "ai_q_018",
                "no": "18",
                "questionNo": "18",
                "type": "subjective",
                "score": 8,
                "standardAnswer": "根据面积公式拆分图形并计算。",
                "scoringRules": ["拆分图形 2 分", "公式正确 2 分", "计算正确 3 分", "单位完整 1 分"],
                "knowledge": ["几何面积"],
                "region": {"page": 2, "x": 110, "y": 640, "width": 600, "height": 160},
            },
        ],
        "reviewRequired": True,
        "source": "worker-mock",
    }


def mock_grade_subjective(payload: dict[str, Any]) -> dict[str, Any]:
    full_score = float(payload.get("fullScore") or payload.get("score") or 10)
    student_answer = str(payload.get("studentAnswer") or payload.get("ocrText") or "")
    standard_answer = str(payload.get("standardAnswer") or "")
    scoring_rules = payload.get("scoringRules") or []
    if not isinstance(scoring_rules, list):
        scoring_rules = []

    evidence = evidence_fragments(student_answer, standard_answer)
    ratio = min(1.0, 0.55 + 0.12 * len(evidence))
    if "x = 24" in student_answer or "24" in student_answer:
        ratio = max(ratio, 0.82)
    score = round(full_score * min(ratio, 0.95), 1)
    confidence = 72 + min(22, len(evidence) * 5)
    return {
        "score": score,
        "fullScore": full_score,
        "reason": "命中关键步骤，需教师复核书写规范和单位完整性。",
        "evidence": evidence,
        "comments": scoring_rules[:3] or ["核心步骤完整", "表达和单位需复核"],
        "confidence": confidence,
        "modelVersion": os.getenv("AI_MODEL_VERSION", "mock-ai-worker-v1"),
    }


def mock_detect_wrong_reason(payload: dict[str, Any]) -> dict[str, Any]:
    answer = str(payload.get("studentAnswer") or payload.get("ocrText") or "")
    knowledge = payload.get("knowledge") or ["比例", "应用题建模"]
    if not isinstance(knowledge, list):
        knowledge = [str(knowledge)]
    if "单位" not in answer and ("24" in answer or "x" in answer):
        error_type = "表达不完整"
        reason = "过程基本正确，但单位或答语不完整。"
    elif any(token in answer for token in ["+", "-", "×", "*"]) and "比例" in "".join(knowledge):
        error_type = "建模错误"
        reason = "未稳定建立比例关系，计算过程偏离题意。"
    else:
        error_type = "概念错误"
        reason = "关键概念或解题路径缺失，需要回到知识点重练。"
    return {
        "wrongReason": reason,
        "errorType": error_type,
        "knowledge": knowledge,
        "trainingHint": "建议安排 5 道同知识点分层练习，并要求补写完整订正过程。",
        "confidence": 82,
    }


def anthropic_messages_url(base_url: str) -> str:
    if base_url.endswith("/messages"):
        return base_url
    if base_url.endswith("/v1"):
        return f"{base_url}/messages"
    return f"{base_url}/v1/messages"


def make_ssl_context() -> ssl.SSLContext:
    default_paths = ssl.get_default_verify_paths()
    if default_paths.cafile and Path(default_paths.cafile).exists():
        return ssl.create_default_context()
    for candidate in ["/etc/ssl/cert.pem", "/etc/ssl/certs/ca-certificates.crt", "/usr/local/etc/openssl@3/cert.pem"]:
        if Path(candidate).exists():
            return ssl.create_default_context(cafile=candidate)
    return ssl.create_default_context()


def parse_ai_json_response(response: dict[str, Any]) -> dict[str, Any]:
    if isinstance(response.get("output_text"), str):
        text = response["output_text"]
    else:
        parts = []
        content = response.get("content")
        if isinstance(content, list):
            for item in content:
                if isinstance(item, dict) and isinstance(item.get("text"), str):
                    parts.append(item["text"])
                elif isinstance(item, str):
                    parts.append(item)
        text = "\n".join(parts)
    data = extract_json_object(text)
    if not isinstance(data, dict):
        raise RuntimeError("AI response is not a JSON object")
    return data


def extract_json_object(text: str) -> dict[str, Any]:
    value = text.strip()
    if value.startswith("```"):
        value = value.strip("`").strip()
        if value.startswith("json"):
            value = value[4:].strip()
    start = value.find("{")
    end = value.rfind("}")
    if start >= 0 and end > start:
        value = value[start : end + 1]
    return json.loads(value)


def normalize_suggested_questions(value: Any, fallback: list[dict[str, Any]]) -> list[dict[str, Any]]:
    if not isinstance(value, list) or not value:
        return fallback
    questions = []
    for index, raw in enumerate(value, start=1):
        if not isinstance(raw, dict):
            continue
        question_no = str(raw.get("questionNo") or raw.get("no") or index)
        questions.append(
            {
                "id": str(raw.get("id") or f"ai_q_{index:03d}"),
                "no": str(raw.get("no") or question_no),
                "questionNo": question_no,
                "type": str(raw.get("type") or "subjective"),
                "score": to_float(raw.get("score"), 0),
                "standardAnswer": str(raw.get("standardAnswer") or ""),
                "scoringRules": normalize_string_list(raw.get("scoringRules"), []),
                "knowledge": normalize_string_list(raw.get("knowledge"), []),
                "region": normalize_region(raw.get("region")),
            }
        )
    return questions or fallback


def normalize_region(value: Any) -> dict[str, Any]:
    if not isinstance(value, dict):
        return {"page": 1, "x": 0, "y": 0, "width": 0, "height": 0}
    return {
        "page": to_int(value.get("page"), 1),
        "x": to_float(value.get("x"), 0),
        "y": to_float(value.get("y"), 0),
        "width": to_float(value.get("width"), 0),
        "height": to_float(value.get("height"), 0),
    }


def normalize_string_list(value: Any, fallback: list[Any]) -> list[str]:
    if isinstance(value, list):
        result = [str(item) for item in value if str(item).strip()]
        return result or [str(item) for item in fallback]
    if isinstance(value, str) and value.strip():
        return [value.strip()]
    return [str(item) for item in fallback]


def to_int(value: Any, default: int) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def to_float(value: Any, default: float) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def clamp_int(value: Any, default: int, minimum: int, maximum: int) -> int:
    return max(minimum, min(maximum, to_int(value, default)))


class ResultWriter:
    def __init__(self, config: WorkerConfig) -> None:
        self.config = config

    def persist_local(self, task_id: str, result: dict[str, Any]) -> Path:
        self.config.result_dir.mkdir(parents=True, exist_ok=True)
        target = self.config.result_dir / f"{task_id}.json"
        target.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
        return target

    def update_status(self, task_id: str, status: str, progress: int, failure_reason: str = "", retry_count: int = 0) -> None:
        payload = {
            "status": status,
            "progress": progress,
            "failureReason": failure_reason,
            "retryCount": retry_count,
        }
        self._request("PATCH", f"/api/scan/tasks/{task_id}/status", payload)

    def write_result(self, task_id: str, result: dict[str, Any]) -> None:
        payload = {
            "status": result.get("status", "识别完成"),
            "progress": result.get("progress", 100),
            "failureReason": result.get("failureReason", ""),
            "modelVersion": self.config.model_version,
            "result": result,
        }
        self._request("POST", f"/api/scan/tasks/{task_id}/worker-result", payload)

    def _request(self, method: str, path: str, payload: dict[str, Any]) -> None:
        if not self.config.api_base_url:
            return
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        request = urllib.request.Request(
            f"{self.config.api_base_url}{path}",
            data=body,
            method=method,
            headers={"Content-Type": "application/json; charset=utf-8"},
        )
        try:
            with urllib.request.urlopen(request, timeout=5) as response:
                response.read()
        except (urllib.error.URLError, TimeoutError) as exc:
            log_event("api_write_failed", method=method, path=path, error=str(exc))


class Worker:
    def __init__(self, config: WorkerConfig) -> None:
        self.config = config
        self.ocr = build_ocr_adapter()
        self.ai = build_ai_adapter(config)
        self.writer = ResultWriter(config)

    def process_scan_task(self, payload: dict[str, Any], callback: bool = True) -> dict[str, Any]:
        start = time.perf_counter()
        task_id = str(payload.get("taskId") or payload.get("id") or f"scan_{int(time.time() * 1000)}")
        log_event("task_started", taskId=task_id, modelVersion=self.config.model_version)
        if callback:
            self.writer.update_status(task_id, "识别中", 10)

        files = normalize_files(payload)
        template = {"templateId": payload.get("templateId"), "templateVersion": payload.get("templateVersion")}
        processed_files = []
        for file_ref in files:
            file_start = time.perf_counter()
            preprocess = preprocess_scan(file_ref)
            try:
                ocr = self.ocr.extract(file_ref)
            except Exception as exc:
                provider = getattr(self.ocr, "name", "unknown")
                file_id = file_ref.get("key") or file_ref.get("url") or file_ref.get("fileName") or "unknown"
                log_event("ocr_extract_failed", file=file_id, provider=provider, error=str(exc))
                if self.config.paddleocr_service_url or provider in {"remote-paddleocr", "paddleocr"}:
                    failure_reason = f"OCR识别失败：{file_id}"
                    if callback:
                        self.writer.update_status(task_id, "识别失败", 0, failure_reason)
                    raise RuntimeError(f"{failure_reason}: {exc}") from exc
                ocr = MockOCRAdapter().extract(file_ref)
            omr = recognize_omr(file_ref, template)
            processed_files.append(
                {
                    "file": file_ref,
                    "preprocess": preprocess,
                    "ocr": ocr,
                    "omr": omr,
                    "durationMs": elapsed_ms(file_start),
                }
            )

        template_suggestion = self.ai.analyze_paper(
            {
                "paperName": payload.get("title") or payload.get("paperName"),
                "sourceFileUrl": files[0].get("url") if files else "",
                "ocrResults": [item["ocr"] for item in processed_files],
                "scanType": payload.get("scanType", ""),
                "templateId": payload.get("templateId", ""),
                "templateVersion": payload.get("templateVersion", 0),
            }
        )
        subjective_results = [
            self.ai.grade_subjective(
                {
                    "fullScore": 10,
                    "standardAnswer": "先设未知数 x，列出比例关系 3:5 = x:40，解得 x = 24。",
                    "studentAnswer": processed_files[0]["ocr"]["text"] if processed_files else "",
                    "scoringRules": ["正确设未知数 2 分", "列出比例关系 4 分", "计算过程正确 2 分", "答语完整 2 分"],
                }
            )
        ]
        wrong_reasons = [
            self.ai.detect_wrong_reason(
                {
                    "studentAnswer": processed_files[0]["ocr"]["text"] if processed_files else "",
                    "knowledge": ["比例", "应用题建模"],
                }
            )
        ]
        result = {
            "taskId": task_id,
            "status": "识别完成",
            "progress": 100,
            "templateId": payload.get("templateId", ""),
            "templateVersion": payload.get("templateVersion", 0),
            "modelVersion": self.config.model_version,
            "ocrResults": [item["ocr"] for item in processed_files],
            "omrResults": [item["omr"] for item in processed_files],
            "preprocessResults": [item["preprocess"] for item in processed_files],
            "templateSuggestion": template_suggestion,
            "subjectiveResults": subjective_results,
            "wrongReasons": wrong_reasons,
            "files": processed_files,
            "durationMs": elapsed_ms(start),
        }
        local_path = self.writer.persist_local(task_id, result)
        result["localResultPath"] = str(local_path)
        if callback:
            self.writer.write_result(task_id, result)
        log_event("task_finished", taskId=task_id, durationMs=result["durationMs"], resultPath=str(local_path))
        return result


class RedisRESPClient:
    def __init__(self, config: WorkerConfig) -> None:
        self.config = config
        self.sock: socket.socket | None = None
        self.reader: Any = None

    def connect(self) -> None:
        host, port = parse_addr(self.config.redis_addr)
        self.sock = socket.create_connection((host, port), timeout=5)
        self.sock.settimeout(30)
        self.reader = self.sock.makefile("rb")
        if self.config.redis_password:
            self.command("AUTH", self.config.redis_password)
        if self.config.redis_db > 0:
            self.command("SELECT", str(self.config.redis_db))

    def close(self) -> None:
        if self.reader is not None:
            self.reader.close()
        if self.sock is not None:
            self.sock.close()

    def command(self, *args: str) -> Any:
        if self.sock is None:
            raise RuntimeError("redis client is not connected")
        body = f"*{len(args)}\r\n" + "".join(f"${len(arg.encode('utf-8'))}\r\n{arg}\r\n" for arg in args)
        self.sock.sendall(body.encode("utf-8"))
        return self.read_response()

    def read_response(self) -> Any:
        line = self.reader.readline()
        if not line:
            raise RuntimeError("redis connection closed")
        prefix = line[:1]
        payload = line[1:].rstrip(b"\r\n")
        if prefix == b"+":
            return payload.decode("utf-8")
        if prefix == b"-":
            raise RuntimeError(payload.decode("utf-8"))
        if prefix == b":":
            return int(payload)
        if prefix == b"$":
            size = int(payload)
            if size < 0:
                return None
            data = self.reader.read(size)
            self.reader.read(2)
            return data.decode("utf-8")
        if prefix == b"*":
            count = int(payload)
            if count < 0:
                return None
            return [self.read_response() for _ in range(count)]
        raise RuntimeError(f"unknown redis response: {line!r}")


class RedisConsumer:
    def __init__(self, config: WorkerConfig, worker: Worker) -> None:
        self.config = config
        self.worker = worker

    def ensure_group(self, client: RedisRESPClient) -> None:
        try:
            client.command("XGROUP", "CREATE", self.config.redis_stream, self.config.redis_group, "0", "MKSTREAM")
        except RuntimeError as exc:
            if "BUSYGROUP" not in str(exc):
                raise

    def consume_once(self, block_ms: int = 1000) -> bool:
        client = RedisRESPClient(self.config)
        client.connect()
        try:
            self.ensure_group(client)
            response = client.command(
                "XREADGROUP",
                "GROUP",
                self.config.redis_group,
                self.config.redis_consumer,
                "COUNT",
                "1",
                "BLOCK",
                str(block_ms),
                "STREAMS",
                self.config.redis_stream,
                ">",
            )
            entries = parse_xread_response(response)
            if not entries:
                return False
            for message_id, fields in entries:
                payload = fields.get("payload")
                task_payload = json.loads(payload) if payload else fields
                self.worker.process_scan_task(task_payload, callback=True)
                client.command("XACK", self.config.redis_stream, self.config.redis_group, message_id)
                log_event("redis_message_acked", messageId=message_id, taskId=task_payload.get("taskId"))
            return True
        finally:
            client.close()

    def run_forever(self) -> None:
        log_event("redis_consumer_started", stream=self.config.redis_stream, group=self.config.redis_group)
        while True:
            try:
                self.consume_once(block_ms=5000)
            except Exception as exc:
                log_event("redis_consumer_error", error=str(exc))
                time.sleep(3)


def redis_ping(config: WorkerConfig) -> dict[str, Any]:
    start = time.perf_counter()
    client = RedisRESPClient(config)
    client.connect()
    try:
        response = client.command("PING")
        return {"status": "ok", "response": response, "latencyMs": elapsed_ms(start)}
    finally:
        client.close()


def parse_xread_response(response: Any) -> list[tuple[str, dict[str, str]]]:
    entries: list[tuple[str, dict[str, str]]] = []
    if not response:
        return entries
    for stream_item in response:
        if not isinstance(stream_item, list) or len(stream_item) < 2:
            continue
        for raw_entry in stream_item[1] or []:
            if not isinstance(raw_entry, list) or len(raw_entry) < 2:
                continue
            message_id = str(raw_entry[0])
            raw_fields = raw_entry[1] or []
            fields: dict[str, str] = {}
            for index in range(0, len(raw_fields), 2):
                if index + 1 < len(raw_fields):
                    fields[str(raw_fields[index])] = str(raw_fields[index + 1])
            entries.append((message_id, fields))
    return entries


def parse_addr(addr: str) -> tuple[str, int]:
    if ":" not in addr:
        return addr, 6379
    host, raw_port = addr.rsplit(":", 1)
    return host, int(raw_port)


def normalize_files(payload: dict[str, Any]) -> list[dict[str, Any]]:
    files = payload.get("files")
    if isinstance(files, list) and files:
        return [file if isinstance(file, dict) else {"key": str(file)} for file in files]
    file_keys = payload.get("fileKeys")
    if isinstance(file_keys, str):
        keys = [item.strip() for item in file_keys.split(",") if item.strip()]
        return [{"key": key, "fileName": Path(key).name, "url": ""} for key in keys]
    if isinstance(file_keys, list):
        return [{"key": str(key), "fileName": Path(str(key)).name, "url": ""} for key in file_keys]
    return [{"key": "mock/scan/student-answer.png", "fileName": "student-answer.png", "url": "/mock/student-answer-q15.png"}]


def evidence_fragments(student_answer: str, standard_answer: str) -> list[str]:
    fragments = []
    for token in ["x", "3:5", "x:40", "24", "答"]:
        if token in student_answer or token in standard_answer:
            fragments.append(token)
    return fragments[:4]


def stable_number(value: str, modulo: int) -> int:
    digest = hashlib.sha256(value.encode("utf-8")).hexdigest()
    return int(digest[:8], 16) % modulo


def load_sample_manifest(config: WorkerConfig) -> dict[str, Any]:
    if not config.sample_manifest.exists():
        return {"samples": []}
    return json.loads(config.sample_manifest.read_text(encoding="utf-8"))


def make_handler(config: WorkerConfig, worker: Worker) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        def do_OPTIONS(self) -> None:
            self.write_json(204, {})

        def do_GET(self) -> None:
            if self.path == "/health":
                self.write_json(200, {"status": "ok", "config": config.public(), "samples": load_sample_manifest(config)})
                return
            if self.path == "/samples/manifest":
                self.write_json(200, load_sample_manifest(config))
                return
            self.write_json(404, {"error": {"code": "NOT_FOUND", "message": "not found"}})

        def do_POST(self) -> None:
            payload = self.read_json()
            if self.path == "/ai/ocr":
                self.write_json(200, worker.ocr.extract(payload))
                return
            if self.path == "/ai/omr":
                self.write_json(200, recognize_omr(payload, payload.get("template") if isinstance(payload.get("template"), dict) else None))
                return
            if self.path == "/ai/paper/analyze":
                self.write_json(200, worker.ai.analyze_paper(payload))
                return
            if self.path == "/ai/grading/subjective":
                self.write_json(200, worker.ai.grade_subjective(payload))
                return
            if self.path == "/ai/wrong-reason":
                self.write_json(200, worker.ai.detect_wrong_reason(payload))
                return
            if self.path == "/worker/process-scan-task":
                self.write_json(200, worker.process_scan_task(payload, callback=env_bool("AI_WORKER_HTTP_CALLBACK", False)))
                return
            if self.path == "/worker/consume-once":
                consumed = RedisConsumer(config, worker).consume_once(block_ms=1000)
                self.write_json(200, {"consumed": consumed})
                return
            self.write_json(404, {"error": {"code": "NOT_FOUND", "message": "not found"}})

        def read_json(self) -> dict[str, Any]:
            length = int(self.headers.get("Content-Length", "0"))
            if length == 0:
                return {}
            try:
                data = json.loads(self.rfile.read(length))
            except json.JSONDecodeError:
                return {}
            return data if isinstance(data, dict) else {}

        def write_json(self, status: int, data: dict[str, Any]) -> None:
            body = b"" if status == 204 else json.dumps(data, ensure_ascii=False).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.send_header("Access-Control-Allow-Headers", "Content-Type")
            self.send_header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            if status != 204:
                self.wfile.write(body)

        def log_message(self, format: str, *args: Any) -> None:
            log_event("http_request", remote=self.client_address[0], message=format % args)

    return Handler


def main() -> None:
    load_dotenv()
    parser = argparse.ArgumentParser(description="AI Worker for OCR, OMR, template split, grading and wrong-reason analysis")
    parser.add_argument("--consume", action="store_true", help="run Redis consumer loop instead of HTTP server")
    parser.add_argument("--consume-once", action="store_true", help="consume at most one Redis task and exit")
    parser.add_argument("--redis-ping", action="store_true", help="connect to Redis and run PING without consuming queue messages")
    parser.add_argument("--process-sample", action="store_true", help="process a built-in sample task and exit")
    args = parser.parse_args()

    config = WorkerConfig.from_env()
    worker = Worker(config)
    if args.process_sample:
        result = worker.process_scan_task({"taskId": "sample_scan_001", "title": "样本扫描任务", "fileKeys": ["samples/student-paper.sample.json"]}, callback=False)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return
    if args.redis_ping:
        print(json.dumps(redis_ping(config), ensure_ascii=False))
        return
    if args.consume_once:
        consumed = RedisConsumer(config, worker).consume_once(block_ms=1000)
        print(json.dumps({"consumed": consumed}, ensure_ascii=False))
        return
    if args.consume:
        RedisConsumer(config, worker).run_forever()
        return

    if config.enable_redis_consumer:
        import threading

        threading.Thread(target=RedisConsumer(config, worker).run_forever, daemon=True).start()

    server = ThreadingHTTPServer(("0.0.0.0", config.port), make_handler(config, worker))
    log_event("http_server_started", url=f"http://localhost:{config.port}", modelVersion=config.model_version)
    server.serve_forever()


if __name__ == "__main__":
    main()
