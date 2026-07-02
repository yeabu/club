# PaddleOCR CPU Service

本目录提供一套本地 PaddleOCR HTTP 服务，目标环境：

- x86_64 Linux
- CPU only
- Docker 或裸机 Python 均可部署
- 对外提供 HTTP OCR 接口，供 Go API、AI Worker 或调试脚本调用

当前 Dockerfile 固定使用 CPU 稳定组合：

- PaddlePaddle 官方 CPU 基础镜像，默认 `paddlepaddle/paddle:2.6.2`
- `paddleocr==2.7.3`

不要在 CPU 基础版里直接安装未固定版本的 `paddleocr`。PaddleOCR 3.x 在部分 x86 CPU 环境下会走 PaddlePaddle PIR/oneDNN 路径，可能触发 `ConvertPirAttribute2RuntimeAttribute` 推理错误。

不要从 `python:slim` 自行 `pip install paddlepaddle` 作为生产方案。已经遇到过 `import paddle` 直接段错误；官方 Paddle 基础镜像的系统库和 wheel 组合更可控。

## 端口

默认监听：

```bash
0.0.0.0:8100
```

## Docker 部署

在服务器上进入项目根目录：

```bash
cd /opt/club/services/paddleocr-service
docker build --no-cache --pull -t club-paddleocr-cpu:latest .
docker run -d \
  --name club-paddleocr \
  --restart unless-stopped \
  -p 8100:8100 \
  -e PADDLEOCR_LANG=ch \
  -e PADDLEOCR_CPU_THREADS=4 \
  -e PADDLEOCR_DISABLE_ONEDNN=true \
  -v /data/club/paddleocr:/data/paddleocr \
  club-paddleocr-cpu:latest
```

如果服务器不能直接拉 Docker Hub，可先把官方镜像同步到内网 Harbor，然后指定构建参数：

```bash
docker build --no-cache \
  --build-arg PADDLE_BASE_IMAGE=a-harbor.wetok168.com/library/paddle:2.6.2 \
  -t club-paddleocr-cpu:latest .
```

健康检查：

```bash
curl http://127.0.0.1:8100/health
```

OCR 调用：

```bash
curl -X POST http://127.0.0.1:8100/ocr \
  -F "file=@/path/to/scan.jpg"
```

第一次调用会下载 PaddleOCR 模型，建议提前执行：

```bash
curl -X POST http://127.0.0.1:8100/warmup
```

确认版本：

```bash
docker exec club-paddleocr python -c "import sys,paddle,paddleocr; print(sys.version); print(paddle.__version__); print(paddleocr.__version__)"
```

如果 `import paddle` 卡住或 `EXIT:139`，说明基础镜像仍不适配当前宿主机。此时不要继续调业务代码，先更换 Paddle 官方 CPU 镜像标签或宿主机运行环境。

## 识别能力边界

这套服务安装了两部分：

- `paddlepaddle`：CPU 推理引擎。
- `paddleocr`：OCR 识别库，HTTP 服务通过 `from paddleocr import PaddleOCR` 调用。

适合优先用于：

- 试卷印刷体文字识别。
- 题号、页眉、学生姓名栏等结构化文字识别。
- 生成答题卡前的试卷版面文字辅助解析。

不建议单独依赖它完成：

- 主观题手写答案的高精度全文转写。
- 潦草、连笔、涂改严重的中文手写识别。
- 答题卡选择题填涂判定。

答题卡填涂属于 OMR，不是普通 OCR。选择题建议用独立的 OMR 规则或视觉检测：定位选项框、二值化、计算填涂面积和置信度，再把低置信度样本送人工复核。

主观题手写答案建议作为“辅助识别 + 教师复核”使用。实际评分链路应优先保留原图，由 AI 给建议分和理由，教师最终确认；不要把 PaddleOCR 的手写转写结果当作唯一评分依据。

## 裸机部署

建议 Python 3.10 或 3.11：

```bash
cd /opt/club/services/paddleocr-service
python3 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip setuptools wheel
python -m pip install paddlepaddle==2.6.2 -i https://www.paddlepaddle.org.cn/packages/stable/cpu/
python -m pip install -r requirements.txt
PADDLEOCR_LANG=ch PADDLEOCR_CPU_THREADS=4 PADDLEOCR_DISABLE_ONEDNN=true uvicorn app:app --host 0.0.0.0 --port 8100
```

## systemd 示例

```ini
[Unit]
Description=Club PaddleOCR CPU Service
After=network.target

[Service]
WorkingDirectory=/opt/club/services/paddleocr-service
Environment=PADDLEOCR_LANG=ch
Environment=PADDLEOCR_CPU_THREADS=4
Environment=PADDLEOCR_DISABLE_ONEDNN=true
Environment=PADDLEOCR_MODEL_DIR=/data/club/paddleocr/models
ExecStart=/opt/club/services/paddleocr-service/.venv/bin/uvicorn app:app --host 0.0.0.0 --port 8100
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PADDLEOCR_LANG` | `ch` | OCR 语言，中文用 `ch` |
| `PADDLEOCR_USE_ANGLE_CLS` | `true` | 是否启用方向分类 |
| `PADDLEOCR_CPU_THREADS` | `4` | CPU 推理线程数 |
| `PADDLEOCR_DISABLE_ONEDNN` | `true` | 禁用 oneDNN/PIR CPU 路径，规避部分 x86 CPU 环境下 PaddlePaddle 3.x 推理错误 |
| `PADDLEOCR_MODEL_DIR` | `/data/paddleocr/models` | PaddleOCR 模型目录 |
| `PADDLEOCR_TMP_DIR` | `/tmp/paddleocr-service` | 上传临时文件目录 |
| `PADDLEOCR_MAX_UPLOAD_MB` | `25` | 单文件上传上限 |

## API

### GET `/health`

返回服务状态。

### POST `/warmup`

预加载 OCR 模型。

### POST `/ocr`

`multipart/form-data` 上传字段：

- `file`: 图片或 PDF 转换后的图片文件

返回：

```json
{
  "provider": "paddleocr",
  "elapsedMs": 1234,
  "text": "识别出的全文",
  "blocks": [
    {
      "text": "单行文本",
      "confidence": 0.98,
      "box": [[0, 0], [100, 0], [100, 30], [0, 30]]
    }
  ]
}
```

### POST `/ocr/base64`

JSON body：

```json
{
  "fileName": "scan.jpg",
  "contentBase64": "..."
}
```

## 服务器建议

基础 CPU 版建议：

- 2 核 CPU / 4 GB RAM：可跑通低并发测试
- 4 核 CPU / 8 GB RAM：更适合教师批量扫描
- 磁盘至少预留 20 GB，用于模型、临时文件和日志

生产环境建议把该服务只暴露在内网，由 AI Worker 或 Go API 调用，不直接开放公网。
