# API Contract Draft

## Endpoints

- `GET /health`
- `GET /api/dashboard`
- `GET /api/grading/subjective/current`
- `GET /api/grading/subjective/reviews/{reviewID}`
- `POST /api/grading/subjective/decision`
- `POST /api/scan/uploads`
- `GET /api/scan/tasks`
- `POST /api/scan/tasks`
- `GET /api/scan/tasks/{taskID}`
- `PATCH /api/scan/tasks/{taskID}/status`
- `POST /api/scan/tasks/{taskID}/retry`
- `POST /api/scan/tasks/{taskID}/match`
- `GET /api/scan/tasks/{taskID}/preview`
- `GET /api/templates`
- `POST /api/templates`
- `GET /api/templates/{templateID}`
- `PUT /api/templates/{templateID}`
- `DELETE /api/templates/{templateID}`
- `POST /api/templates/{templateID}/copy`
- `PUT /api/templates/{templateID}/status`
- `POST /api/templates/{templateID}/ai-suggestions`
- `GET /api/templates/{templateID}/regions`
- `PUT /api/templates/{templateID}/regions`
- `POST /api/templates/{templateID}/regions`
- `PUT /api/templates/{templateID}/regions/{regionID}`
- `DELETE /api/templates/{templateID}/regions/{regionID}`
- `GET /api/analytics/classroom`
- `GET /api/dev/connections`
- `POST /api/dev/reset-demo`

## Dashboard

`GET /api/dashboard` returns teacher workspace metrics and queues. `source` is `database` when the Go API reads real database data, and `fixtures` when the Go API is available but has fallen back to demo data.

```json
{
  "source": "database",
  "metrics": [
    { "label": "待批试卷", "value": "12", "delta": "来自扫描队列", "tone": "primary" }
  ],
  "scanQueue": [],
  "reviewQueue": [],
  "weakPoints": [],
  "homeworkWatch": []
}
```

## Scan Import

`POST /api/scan/uploads` accepts multipart form files in the `files` field. Supported formats are PDF, PNG, JPG, WebP, and ZIP scan packages. Each file is limited to 25 MB. The development implementation writes files to the API upload store and returns object-style keys that can later map to MinIO or OBS.

```json
{
  "files": [
    {
      "key": "scan/20260619/1781820000000-paper.pdf",
      "fileName": "paper.pdf",
      "contentType": "application/pdf",
      "size": 245760,
      "url": "/uploads/scan/20260619/1781820000000-paper.pdf"
    }
  ]
}
```

`GET /api/scan/tasks` returns recent scan tasks for polling. `GET /api/scan/tasks/{taskID}` returns a single task. Web polls these endpoints to update `status`, `progress`, `failureReason`, and queue delivery state. `PATCH /api/scan/tasks/{taskID}/status` lets the Worker update processing state.

`POST /api/scan/tasks` creates a scan OCR queue task after upload and writes a Redis Stream message to `club:scan:tasks`. The task must bind a `published` template. The API stores that template's current `version` on the task so later template edits do not change historical scan processing. The stream fields include `taskId`, `templateId`, `templateVersion`, `fileKeys`, and a JSON `payload` for the Worker.

```json
{
  "title": "六年级数学期中卷",
  "className": "六年级 3 班",
  "templateId": "tpl_001",
  "templateVersion": 1,
  "pages": 48,
  "notes": "期中考试整班扫描",
  "files": [
    {
      "key": "scan/20260619/1781820000000-paper.pdf",
      "fileName": "paper.pdf",
      "contentType": "application/pdf",
      "size": 245760,
      "url": "/uploads/scan/20260619/1781820000000-paper.pdf"
    }
  ]
}
```

```json
{
  "status": "created",
  "task": {
    "id": "scan_1781820000000",
    "title": "六年级数学期中卷",
    "className": "六年级 3 班",
    "templateId": "tpl_001",
    "templateVersion": 1,
    "pages": 48,
    "notes": "期中考试整班扫描",
    "status": "排队中",
    "progress": 0,
    "failureReason": "",
    "retryCount": 0,
    "queueStatus": "queued",
    "queueMessage": "1781820000000-0",
    "files": []
  }
}
```

Worker status update:

```json
{
  "status": "识别中",
  "progress": 45,
  "failureReason": "",
  "retryCount": 0
}
```

Retry one file or a whole task:

```json
{
  "fileKey": "scan/20260619/1781820000000-paper.pdf"
}
```

Omit `fileKey` to retry the whole batch. The API increments `retryCount`, clears failure state, logs the retry, and writes the task back to Redis.

Manual student matching:

```json
{
  "fileKey": "scan/20260619/1781820000000-paper.pdf",
  "studentId": "stu_001",
  "studentName": "张三",
  "matchMethod": "manual"
}
```

`GET /api/scan/tasks/{taskID}/preview` returns the task and its files with `url`, `page`, `status`, `studentName`, `matchStatus`, and per-file `failureReason` so Web can preview original scans before grading.

## AI Template Suggestions

`POST /api/templates/{templateID}/ai-suggestions` returns Worker-style paper split suggestions for a saved draft template. Web loads these regions into the template canvas; the teacher confirms them by calling `PUT /api/templates/{templateID}/regions`.

```json
{
  "paperName": "六年级数学期中卷",
  "sourceFileUrl": "/uploads/scan/20260619/blank.pdf"
}
```

```json
{
  "paperName": "六年级数学期中卷",
  "questionCount": 3,
  "totalScore": 20,
  "reviewRequired": true,
  "source": "worker-mock",
  "suggestedQuestions": [
    {
      "id": "ai_q_001",
      "no": "1",
      "type": "single_choice",
      "score": 2,
      "standardAnswer": "A",
      "scoringRules": ["选对 A 得 2 分"],
      "knowledge": ["分数"],
      "region": { "page": 1, "x": 120, "y": 260, "width": 480, "height": 80 }
    }
  ]
}
```

## Subjective Grading Decision

```json
{
  "submissionId": "sub_001",
  "questionId": "q_015",
  "finalScore": 8,
  "decision": "accepted_ai",
  "teacherNote": "步骤完整，结果计算错误，按规则扣 2 分"
}
```

## Subjective Grading Response

```json
{
  "reviewId": "review_001",
  "submissionId": "sub_001",
  "questionId": "q_015",
  "paperName": "六年级数学期中卷",
  "studentName": "张三",
  "questionNo": "15",
  "fullScore": 10
}
```

## Decision Response

```json
{
  "status": "saved",
  "finalScore": 8.5,
  "nextQuestion": "q_018",
  "nextReview": {
    "reviewId": "review_002",
    "questionId": "q_018"
  }
}
```

## Paper Template

`POST /api/templates` creates a draft template. `PUT /api/templates/{templateID}` replaces template metadata and question regions only while the template is `draft`. `POST /api/templates/{templateID}/copy` creates a new draft version. `PUT /api/templates/{templateID}/status` moves a template between `draft`, `published`, and `disabled`.

```json
{
  "id": "tpl_001",
  "name": "六年级数学期中卷",
  "subject": "数学",
  "grade": "六年级",
  "questionCount": 2,
  "totalScore": 12,
  "sourceFileUrl": "/mock/templates/tpl_001-blank-paper.pdf",
  "status": "draft",
  "version": 2,
  "parentId": "tpl_001",
  "questions": [
    {
      "id": "q_001",
      "no": "1",
      "type": "single_choice",
      "score": 2,
      "standardAnswer": "A",
      "scoringRules": ["选对 A 得 2 分"],
      "knowledge": ["分数"],
      "region": { "page": 1, "x": 120, "y": 260, "width": 480, "height": 80 }
    }
  ]
}
```

## Template Mutation Response

```json
{
  "status": "created",
  "template": {
    "id": "tpl_001",
    "name": "六年级数学期中卷",
    "subject": "数学",
    "grade": "六年级",
    "questionCount": 2,
    "totalScore": 12,
    "sourceFileUrl": "/mock/templates/tpl_001-blank-paper.pdf",
    "status": "draft",
    "version": 1,
    "parentId": "",
    "questions": []
  }
}
```

## Template Status Request

```json
{
  "status": "published"
}
```

Status values:

- `draft`: editable template version
- `published`: immutable version that can be bound by exams or assignments
- `disabled`: hidden from normal use, kept for history and audit

## Template Region APIs

`GET /api/templates/{templateID}/regions` returns the persisted question regions for a template. `POST` creates one region, `PUT /regions/{regionID}` updates metadata and coordinates, `DELETE` removes one region, and `PUT /regions` replaces the template's regions in one batch.

```json
{
  "id": "q_015",
  "no": "15",
  "type": "subjective",
  "score": 10,
  "standardAnswer": "先设未知数 x，列出比例关系求解。",
  "scoringRules": ["正确设未知数 2 分", "列式 4 分", "计算 2 分", "答语 2 分"],
  "knowledge": ["比例", "应用题建模"],
  "region": { "page": 2, "x": 96, "y": 420, "width": 620, "height": 180 }
}
```

Batch save request:

```json
{
  "questions": [
    {
      "id": "q_015",
      "no": "15",
      "type": "subjective",
      "score": 10,
      "standardAnswer": "先设未知数 x，列出比例关系求解。",
      "scoringRules": ["正确设未知数 2 分", "列式 4 分", "计算 2 分", "答语 2 分"],
      "knowledge": ["比例", "应用题建模"],
      "region": { "page": 2, "x": 96, "y": 420, "width": 620, "height": 180 }
    }
  ]
}
```

Single-region mutation response:

```json
{
  "status": "updated",
  "question": {
    "id": "q_015",
    "no": "15",
    "type": "subjective",
    "score": 10,
    "standardAnswer": "先设未知数 x，列出比例关系求解。",
    "scoringRules": ["正确设未知数 2 分", "列式 4 分"],
    "knowledge": ["比例"],
    "region": { "page": 2, "x": 120, "y": 430, "width": 620, "height": 180 }
  },
  "template": {
    "id": "tpl_001",
    "name": "六年级数学期中卷",
    "subject": "数学",
    "grade": "六年级",
    "questionCount": 25,
    "totalScore": 100,
    "questions": []
  }
}
```
