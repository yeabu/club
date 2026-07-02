# AI Worker

Python 服务负责 OCR、OMR、模板拆解、主观题建议分、错因分析、Redis 队列消费和结果回写。当前实现只依赖 Python 标准库；真实 PaddleOCR 可通过环境变量启用，未安装时自动回退到稳定 mock 适配器。

## 启动

HTTP 调试服务：

```bash
python3 -m app.main
```

处理内置样本并落盘：

```bash
python3 -m app.main --process-sample
```

消费 Redis Stream：

```bash
python3 -m app.main --consume
```

消费一次后退出，适合本地验证：

```bash
python3 -m app.main --consume-once
```

只验证 Redis 连通性，不读取或确认队列消息：

```bash
python3 -m app.main --redis-ping
```

## 配置

- `AI_WORKER_PORT`：HTTP 端口，默认 `8090`
- `AI_WORKER_CONSUME_REDIS`：HTTP 服务启动时是否后台消费 Redis
- `AI_WORKER_SCAN_STREAM`：扫描任务 Stream，默认 `club:scan:tasks`
- `AI_WORKER_REDIS_GROUP`：消费组，默认 `club-ai-worker`
- `API_BASE_URL`：Go API 地址，默认 `http://localhost:8080`
- `AI_WORKER_RESULT_DIR`：本地结果落盘目录，默认 `var/ai-worker/results`
- `AI_MODEL_VERSION`：日志和回写中的模型版本
- `AI_PROVIDER`：大模型厂商，当前支持 `anthropic` / `anthropic-compatible` / `deepseek` / `mock`
- `ANTHROPIC_BASE_URL`：Anthropic Messages 兼容接口地址，例如 `https://api.deepseek.com/anthropic`
- `ANTHROPIC_AUTH_TOKEN`：Anthropic Messages 兼容接口密钥
- `ANTHROPIC_MODEL`：模型名称，例如 `deepseek-v4-pro`
- `AI_TIMEOUT_SECONDS`：大模型 HTTP 调用超时，默认 `45`
- `PADDLEOCR_SERVICE_URL`：远程 PaddleOCR HTTP 服务地址，例如 `http://192.168.0.213:8100`
- `AI_WORKER_USE_PADDLEOCR`：设为 `true` 时优先尝试 PaddleOCR，失败回退 mock OCR

OCR 优先级：

1. 设置 `PADDLEOCR_SERVICE_URL` 时，优先调用远程 PaddleOCR HTTP 服务。
2. 未设置远程服务且 `AI_WORKER_USE_PADDLEOCR=true` 时，尝试本进程加载 PaddleOCR。
3. 以上不可用时回退到稳定 mock OCR，保证队列消费链路不中断。

AI Provider 优先级：

1. 设置 `ANTHROPIC_AUTH_TOKEN` 且 `AI_PROVIDER` 为空、`anthropic`、`anthropic-compatible` 或 `deepseek` 时，调用 Anthropic Messages 兼容接口。
2. 未设置三方接口密钥时，使用本地 mock AI，保证开发和单测不依赖外网。
3. 三方接口返回异常或非 JSON 时，会记录 `ai_provider_failed` 日志并回退 mock 结果，扫描链路不中断。

当前三方 AI 任务覆盖：

- `POST /ai/paper/analyze`：试卷解析并生成答题卡题区建议
- `POST /ai/grading/subjective`：主观题建议分、理由和证据
- `POST /ai/wrong-reason`：错因归类和训练建议

## Endpoints

- `GET /health`
- `GET /samples/manifest`
- `POST /ai/ocr`
- `POST /ai/omr`
- `POST /ai/paper/analyze`
- `POST /ai/grading/subjective`
- `POST /ai/wrong-reason`
- `POST /worker/process-scan-task`
- `POST /worker/consume-once`

## 队列与回写

Worker 监听 Go API 投递的 Redis Stream `club:scan:tasks`。消息字段包含 `taskId`、`templateId`、`templateVersion`、`fileKeys` 和 JSON `payload`。

处理流程：

1. 更新 Go API 扫描任务状态为 `识别中`
2. 对每个文件执行预处理、OCR、OMR
3. 生成模板拆解建议、主观题建议分、错因分析
4. 将完整结果写入 `AI_WORKER_RESULT_DIR`
5. `POST /api/scan/tasks/{taskID}/worker-result` 回写 Go API
6. Redis `XACK` 确认消息

中间件可用性验证优先使用 `--redis-ping`，避免误消费共享队列中的真实任务。

## 样本集

`samples/manifest.json` 管理四类回归样本：

- 空白卷：模板拆解和题区建议
- 答题卡：OMR 客观题识别
- 学生卷：OCR、主观题建议分和错因分析
- 异常扫描件：旋转、低对比度等预处理诊断

## 测试

```bash
python3 -m unittest discover -s tests
```
