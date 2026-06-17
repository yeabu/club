from __future__ import annotations

import json
import os
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


def public_config() -> dict[str, Any]:
    return {
        "appEnv": os.getenv("APP_ENV", "development"),
        "port": int(os.getenv("AI_WORKER_PORT", "8090")),
        "storageDriver": os.getenv("STORAGE_DRIVER", "minio"),
        "mysql": {
            "host": os.getenv("MYSQL_HOST", "127.0.0.1"),
            "port": os.getenv("MYSQL_PORT", "3306"),
            "user": os.getenv("MYSQL_USER", "root"),
            "database": os.getenv("MYSQL_DATABASE", "club"),
            "passwordProvided": bool(os.getenv("MYSQL_PASSWORD")),
        },
        "redis": {
            "addr": os.getenv("REDIS_ADDR", "127.0.0.1:6379"),
            "db": int(os.getenv("REDIS_DB", "0")),
            "passwordProvided": bool(os.getenv("REDIS_PASSWORD")),
        },
        "obs": {
            "endpoint": os.getenv("OBS_ENDPOINT", ""),
            "bucket": os.getenv("OBS_BUCKET", ""),
            "region": os.getenv("OBS_REGION", ""),
            "accessKeyProvided": bool(os.getenv("OBS_ACCESS_KEY_ID")),
            "secretProvided": bool(os.getenv("OBS_SECRET_ACCESS_KEY")),
        },
    }


def analyze_paper(payload: dict[str, Any]) -> dict[str, Any]:
    return {
        "paperName": payload.get("paperName", "未命名试卷"),
        "questionCount": 25,
        "totalScore": 100,
        "suggestedQuestions": [
            {
                "questionNo": "1",
                "type": "single_choice",
                "score": 2,
                "knowledge": ["分数"],
                "region": {"page": 1, "x": 120, "y": 260, "width": 480, "height": 80},
            },
            {
                "questionNo": "15",
                "type": "subjective",
                "score": 10,
                "knowledge": ["比例", "应用题建模"],
                "region": {"page": 2, "x": 96, "y": 420, "width": 620, "height": 180},
            },
        ],
        "reviewRequired": True,
    }


def grade_subjective(payload: dict[str, Any]) -> dict[str, Any]:
    return {
        "score": 8,
        "fullScore": payload.get("fullScore", 10),
        "reason": "建模和计算结果正确，但比例式书写不够规范，建议教师复核。",
        "comments": ["核心步骤完整", "比例式表达需规范", "答语完整"],
        "confidence": 86,
    }


def detect_wrong_reason(payload: dict[str, Any]) -> dict[str, Any]:
    return {
        "wrongReason": "比例关系理解不稳定",
        "knowledge": ["比例", "应用题建模"],
        "trainingHint": "建议安排 5 道比例建模题和 3 道单位换算题。",
    }


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        if self.path == "/health":
            self.write_json(200, {"status": "ok", "config": public_config()})
            return
        self.write_json(404, {"error": "not found"})

    def do_POST(self) -> None:
        payload = self.read_json()
        if self.path == "/ai/paper/analyze":
            self.write_json(200, analyze_paper(payload))
            return
        if self.path == "/ai/grading/subjective":
            self.write_json(200, grade_subjective(payload))
            return
        if self.path == "/ai/wrong-reason":
            self.write_json(200, detect_wrong_reason(payload))
            return
        self.write_json(404, {"error": "not found"})

    def read_json(self) -> dict[str, Any]:
        length = int(self.headers.get("Content-Length", "0"))
        if length == 0:
            return {}
        try:
            return json.loads(self.rfile.read(length))
        except json.JSONDecodeError:
            return {}

    def write_json(self, status: int, data: dict[str, Any]) -> None:
        body = json.dumps(data, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


def main() -> None:
    load_dotenv()
    port = int(os.getenv("AI_WORKER_PORT", "8090"))
    server = ThreadingHTTPServer(("0.0.0.0", port), Handler)
    print(f"ai worker listening on http://localhost:{port}")
    server.serve_forever()


if __name__ == "__main__":
    main()
