# AI Worker

Python 服务负责与 OCR、OMR、LLM 和试卷拆解能力集成。当前版本提供本地可运行的标准库 HTTP 服务和 mock 推理结果，方便前端与 Go API 先联调。

## 启动

```bash
python3 -m app.main
```

## Endpoints

- `GET /health`
- `POST /ai/paper/analyze`
- `POST /ai/grading/subjective`
- `POST /ai/wrong-reason`

