# AI Worker 容器打包部署文档

更新日期：2026-07-02

本文档用于将 `services/ai-worker` 打包为 Docker 镜像并部署到服务器。当前 AI Worker 负责 OCR/OMR 调度、模板拆解建议、主观题建议分、错因分析、Redis 队列消费和结果回写。

## 1. 部署架构

推荐把 AI Worker 和 PaddleOCR 拆成两个容器：

```text
Go API
  -> Redis Stream: club:scan:tasks
  -> AI Worker
      -> PaddleOCR HTTP Service, optional
      -> Third-party AI Provider, optional
      -> Go API callback: /api/scan/tasks/{taskID}/worker-result
```

原因：

- AI Worker 当前只依赖 Python 标准库，镜像可以很小。
- PaddleOCR 依赖 PaddlePaddle 和系统库，镜像较大，CPU 兼容性也更敏感。
- 拆开后 Worker 可以独立扩容，OCR 服务可以按机器性能单独调整。

## 2. 目录约定

项目根目录：

```bash
/opt/club
```

相关目录：

```text
services/ai-worker/          # AI Worker 源码
services/paddleocr-service/  # 可选 PaddleOCR HTTP 服务
var/ai-worker/results/       # Worker 识别结果落盘目录
```

服务器数据目录建议：

```bash
/data/club/ai-worker/results
/data/club/paddleocr
```

## 3. 准备 Dockerfile

在 `services/ai-worker/Dockerfile` 放置以下内容：

```dockerfile
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
```

`.dockerignore` 建议：

```gitignore
__pycache__/
*.pyc
.pytest_cache/
.venv/
var/
```

说明：

- 当前 Worker 不需要 `requirements.txt`。
- 如果后续把第三方 SDK 加入 Worker，再补 `COPY requirements.txt .` 和 `pip install -r requirements.txt`。
- 不建议在这个镜像中直接安装 PaddleOCR；优先使用 `PADDLEOCR_SERVICE_URL` 调远程 OCR 容器。

## 4. 构建镜像

在项目根目录执行：

```bash
cd /opt/club
docker build -t club-ai-worker:20260702 -f services/ai-worker/Dockerfile services/ai-worker
docker tag club-ai-worker:20260702 club-ai-worker:latest
```

验证镜像：

```bash
docker run --rm club-ai-worker:latest python -m app.main --process-sample
```

如果需要推送到镜像仓库：

```bash
docker tag club-ai-worker:20260702 registry.example.com/club/ai-worker:20260702
docker push registry.example.com/club/ai-worker:20260702
```

## 5. 环境变量

| 变量 | 示例 | 说明 |
| --- | --- | --- |
| `APP_ENV` | `production` | 运行环境 |
| `AI_WORKER_PORT` | `8090` | HTTP 服务端口 |
| `AI_WORKER_CONSUME_REDIS` | `true` | HTTP 服务启动后是否同时后台消费 Redis |
| `AI_WORKER_SCAN_STREAM` | `club:scan:tasks` | Redis Stream 名称 |
| `AI_WORKER_REDIS_GROUP` | `club-ai-worker` | Redis 消费组 |
| `AI_WORKER_REDIS_CONSUMER` | `ai-worker-01` | 消费者名称，建议每个实例唯一 |
| `REDIS_ADDR` | `redis:6379` | Redis 地址 |
| `REDIS_DB` | `0` | Redis DB |
| `REDIS_PASSWORD` | `******` | Redis 密码，没有则留空 |
| `API_BASE_URL` | `http://api:8080` | Go API 内网地址 |
| `AI_WORKER_RESULT_DIR` | `/data/ai-worker/results` | 结果落盘目录 |
| `AI_MODEL_VERSION` | `ai-worker-20260702` | 回写和日志中的模型版本 |
| `PADDLEOCR_SERVICE_URL` | `http://paddleocr:8100` | 可选，远程 PaddleOCR 服务 |
| `AI_WORKER_USE_PADDLEOCR` | `false` | 不推荐在 Worker 镜像内加载 PaddleOCR |
| `AI_PROVIDER` | `deepseek` | 可选，三方 AI Provider 类型 |
| `ANTHROPIC_BASE_URL` | `https://api.example.com/anthropic` | Anthropic Messages 兼容接口 |
| `ANTHROPIC_AUTH_TOKEN` | `******` | 三方模型密钥 |
| `ANTHROPIC_MODEL` | `deepseek-v4-pro` | 模型名称 |
| `AI_TIMEOUT_SECONDS` | `45` | 三方 AI 调用超时 |

生产环境不要把密钥写进镜像。用 `.env`、Docker secret、Kubernetes Secret 或部署平台的密钥管理注入。

## 6. Docker Compose 部署

### 6.1 单容器模式

HTTP 服务和 Redis 消费在同一个容器内运行，适合首版部署：

```yaml
services:
  ai-worker:
    image: club-ai-worker:20260702
    container_name: club-ai-worker
    restart: unless-stopped
    ports:
      - "8090:8090"
    environment:
      APP_ENV: production
      AI_WORKER_PORT: "8090"
      AI_WORKER_CONSUME_REDIS: "true"
      AI_WORKER_SCAN_STREAM: club:scan:tasks
      AI_WORKER_REDIS_GROUP: club-ai-worker
      AI_WORKER_REDIS_CONSUMER: ai-worker-01
      REDIS_ADDR: redis:6379
      REDIS_DB: "0"
      REDIS_PASSWORD: ""
      API_BASE_URL: http://api:8080
      AI_WORKER_RESULT_DIR: /data/ai-worker/results
      AI_MODEL_VERSION: ai-worker-20260702
      PADDLEOCR_SERVICE_URL: http://paddleocr:8100
    volumes:
      - /data/club/ai-worker/results:/data/ai-worker/results
    networks:
      - club

networks:
  club:
    external: true
```

启动：

```bash
docker compose -f docker-compose.ai-worker.yml up -d
```

### 6.2 HTTP 与消费者拆分模式

更推荐生产使用。HTTP 服务只提供调试和健康检查，消费者容器专门消费 Redis：

```yaml
services:
  ai-worker-http:
    image: club-ai-worker:20260702
    container_name: club-ai-worker-http
    restart: unless-stopped
    ports:
      - "8090:8090"
    environment:
      APP_ENV: production
      AI_WORKER_PORT: "8090"
      AI_WORKER_CONSUME_REDIS: "false"
      REDIS_ADDR: redis:6379
      API_BASE_URL: http://api:8080
      PADDLEOCR_SERVICE_URL: http://paddleocr:8100
      AI_WORKER_RESULT_DIR: /data/ai-worker/results
    volumes:
      - /data/club/ai-worker/results:/data/ai-worker/results
    networks:
      - club

  ai-worker-consumer:
    image: club-ai-worker:20260702
    container_name: club-ai-worker-consumer
    restart: unless-stopped
    command: ["python", "-m", "app.main", "--consume"]
    environment:
      APP_ENV: production
      AI_WORKER_SCAN_STREAM: club:scan:tasks
      AI_WORKER_REDIS_GROUP: club-ai-worker
      AI_WORKER_REDIS_CONSUMER: ai-worker-consumer-01
      REDIS_ADDR: redis:6379
      REDIS_DB: "0"
      REDIS_PASSWORD: ""
      API_BASE_URL: http://api:8080
      AI_WORKER_RESULT_DIR: /data/ai-worker/results
      AI_MODEL_VERSION: ai-worker-20260702
      PADDLEOCR_SERVICE_URL: http://paddleocr:8100
    volumes:
      - /data/club/ai-worker/results:/data/ai-worker/results
    networks:
      - club

networks:
  club:
    external: true
```

扩容消费者：

```bash
docker compose -f docker-compose.ai-worker.yml up -d --scale ai-worker-consumer=3
```

扩容时要保证 `AI_WORKER_REDIS_CONSUMER` 唯一。Compose 副本场景可改为由宿主机或编排平台注入唯一名称。

## 7. PaddleOCR 容器联动

如果使用远程 OCR，先部署 `services/paddleocr-service`：

```bash
cd /opt/club/services/paddleocr-service
docker build -t club-paddleocr-cpu:20260702 .
docker run -d \
  --name club-paddleocr \
  --restart unless-stopped \
  -p 8100:8100 \
  -e PADDLEOCR_LANG=ch \
  -e PADDLEOCR_CPU_THREADS=4 \
  -e PADDLEOCR_DISABLE_ONEDNN=true \
  -v /data/club/paddleocr:/data/paddleocr \
  club-paddleocr-cpu:20260702
```

健康检查：

```bash
curl http://127.0.0.1:8100/health
curl -X POST http://127.0.0.1:8100/warmup
```

AI Worker 中配置：

```bash
PADDLEOCR_SERVICE_URL=http://paddleocr:8100
```

如果不配置 `PADDLEOCR_SERVICE_URL`，Worker 会回退到 mock OCR，队列链路仍可跑通，但识别结果不适合生产使用。

## 8. 上线验证

### 8.1 健康检查

```bash
curl http://127.0.0.1:8090/health
```

期望返回包含：

```json
{
  "status": "ok",
  "config": {
    "redis": {
      "stream": "club:scan:tasks",
      "group": "club-ai-worker"
    },
    "api": {
      "baseUrl": "http://api:8080"
    }
  }
}
```

### 8.2 Redis 连通性

只验证 Redis，不消费消息：

```bash
docker exec club-ai-worker python -m app.main --redis-ping
```

期望：

```json
{"ok": true, "message": "PONG"}
```

### 8.3 样本任务

```bash
docker exec club-ai-worker python -m app.main --process-sample
```

检查结果目录：

```bash
ls -lh /data/club/ai-worker/results
```

### 8.4 消费一次

适合联调，不建议在共享生产队列随意执行：

```bash
docker exec club-ai-worker python -m app.main --consume-once
```

## 9. 日志与排障

查看日志：

```bash
docker logs -f club-ai-worker
docker logs -f club-ai-worker-consumer
```

关键日志事件：

| 事件 | 含义 |
| --- | --- |
| `http_server_started` | HTTP 服务启动成功 |
| `redis_consumer_started` | Redis 消费者启动成功 |
| `scan_task_started` | 开始处理扫描任务 |
| `stage_completed` | 某处理阶段完成 |
| `worker_result_written` | 本地结果已落盘 |
| `api_callback_succeeded` | 已回写 Go API |
| `ai_provider_failed` | 三方 AI 调用失败，通常会回退 mock |

常见问题：

| 问题 | 排查 |
| --- | --- |
| `/health` 不通 | 检查端口映射、容器日志、`AI_WORKER_PORT` |
| Redis 连接失败 | 检查 `REDIS_ADDR`、网络、密码、Redis ACL |
| 消息不消费 | 检查 `AI_WORKER_CONSUME_REDIS` 或消费者容器 command |
| 回写 API 失败 | 检查 `API_BASE_URL` 是否为容器内可访问地址 |
| OCR 结果都是 mock | 检查 `PADDLEOCR_SERVICE_URL` 和 PaddleOCR `/health` |
| 结果目录无文件 | 检查 volume 挂载和 `AI_WORKER_RESULT_DIR` 写权限 |

## 10. 升级与回滚

构建新版本：

```bash
docker build -t club-ai-worker:20260703 -f services/ai-worker/Dockerfile services/ai-worker
```

更新 Compose 镜像标签：

```yaml
image: club-ai-worker:20260703
```

滚动重启：

```bash
docker compose -f docker-compose.ai-worker.yml up -d
```

回滚：

```yaml
image: club-ai-worker:20260702
```

```bash
docker compose -f docker-compose.ai-worker.yml up -d
```

回滚前确认：

- Redis Stream 中未确认消息数量。
- Go API 与 Worker 的结果字段兼容。
- `AI_MODEL_VERSION` 是否需要随镜像版本回退。

## 11. 生产注意事项

- Worker、Redis、Go API、PaddleOCR 应部署在同一内网，不直接暴露 Redis 和 PaddleOCR 到公网。
- 三方 AI 密钥不得写入镜像和 Git 仓库。
- `AI_WORKER_RESULT_DIR` 要挂载持久化目录，便于问题追溯。
- 多实例消费 Redis Stream 时，消费者名称必须唯一。
- 生产环境建议拆分 HTTP 服务和消费者服务，避免健康检查或调试请求影响消费稳定性。
- OCR、AI Provider 失败时 Worker 会尽量回退 mock 结果；生产环境应通过日志和任务状态监控识别这种降级，避免把 mock 结果当正式识别结果。
