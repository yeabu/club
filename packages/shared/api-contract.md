# API Contract Draft

## Endpoints

- `GET /health`
- `GET /api/dashboard`
- `GET /api/grading/subjective/current`
- `GET /api/grading/subjective/reviews`
- `GET /api/grading/subjective/reviews/{reviewID}`
- `GET /api/grading/subjective/history`
- `POST /api/grading/subjective/decision`
- `POST /api/scan/uploads`
- `GET /api/scan/tasks`
- `POST /api/scan/tasks`
- `GET /api/scan/tasks/{taskID}`
- `PATCH /api/scan/tasks/{taskID}/status`
- `GET /api/scan/tasks/{taskID}/worker-result`
- `POST /api/scan/tasks/{taskID}/worker-result`
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

Worker result write-back:

`POST /api/scan/tasks/{taskID}/worker-result` stores the latest OCR/OMR/AI output for a scan task and updates the task status/progress. `GET /api/scan/tasks/{taskID}/worker-result` returns the task plus the latest stored Worker result.

```json
{
  "status": "识别完成",
  "progress": 100,
  "failureReason": "",
  "modelVersion": "mock-ai-worker-v1",
  "result": {
    "taskId": "scan_1781820000000",
    "templateId": "tpl_001",
    "templateVersion": 1,
    "ocrResults": [
      {
        "provider": "mock-ocr",
        "objectKey": "scan/20260619/1781820000000-paper.pdf",
        "text": "设需要 x 千克，3/5 = x/40，5x = 120，x = 24。",
        "confidence": 91,
        "blocks": []
      }
    ],
    "omrResults": [
      {
        "provider": "mock-omr",
        "answers": [
          { "questionNo": "1", "selected": "A", "confidence": 98 }
        ]
      }
    ],
    "preprocessResults": [
      {
        "objectKey": "scan/20260619/1781820000000-paper.pdf",
        "rotationApplied": 1,
        "grayscale": true,
        "denoise": true,
        "outputKey": "processed/scan/20260619/1781820000000-paper.pdf"
      }
    ],
    "templateSuggestion": {
      "paperName": "六年级数学期中卷",
      "questionCount": 25,
      "totalScore": 100,
      "suggestedQuestions": []
    },
    "subjectiveResults": [
      {
        "score": 8.2,
        "fullScore": 10,
        "reason": "命中关键步骤，需教师复核书写规范和单位完整性。",
        "evidence": ["x", "24"],
        "confidence": 86
      }
    ],
    "wrongReasons": [
      {
        "errorType": "表达不完整",
        "wrongReason": "过程基本正确，但单位或答语不完整。",
        "knowledge": ["比例", "应用题建模"],
        "trainingHint": "建议安排 5 道同知识点分层练习。"
      }
    ],
    "durationMs": 320
  }
}
```

Response:

```json
{
  "status": "saved",
  "task": {
    "id": "scan_1781820000000",
    "status": "识别完成",
    "progress": 100
  },
  "result": {
    "taskId": "scan_1781820000000",
    "status": "识别完成",
    "progress": 100,
    "modelVersion": "mock-ai-worker-v1",
    "result": {}
  }
}
```

Retry one file or a whole task:

```json
{
  "fileKey": "scan/20260619/1781820000000-paper.pdf"
}
```

Omit `fileKey` to retry the whole batch. The API increments `retryCount`, clears failure state, logs the retry, and writes the task back to Redis.

## AI Worker Contract

The Python Worker can run as an HTTP service or Redis consumer.

Worker HTTP endpoints:

- `GET /health`: returns public config, Redis stream/group, model version, and sample manifest summary.
- `GET /samples/manifest`: returns regression sample metadata.
- `POST /ai/ocr`: OCR adapter output with text blocks and confidence.
- `POST /ai/omr`: objective answer recognition output with selected options and confidence.
- `POST /ai/paper/analyze`: blank paper split suggestions for template review.
- `POST /ai/grading/subjective`: subjective suggested score, reason, evidence fragments, comments, confidence.
- `POST /ai/wrong-reason`: error type, wrong reason, knowledge points, training hint, confidence.
- `POST /worker/process-scan-task`: runs the full scan pipeline for one task payload.
- `POST /worker/consume-once`: consumes at most one Redis Stream message.

Redis consumer:

- Stream: `club:scan:tasks`
- Group: `club-ai-worker`
- Message fields: `taskId`, `templateId`, `templateVersion`, `fileKeys`, `payload`
- Processing stages: `preprocess`, `ocr`, `omr`, `templateSuggestion`, `subjectiveGrading`, `wrongReason`, `writeBack`
- A successful run writes local JSON output and calls `POST /api/scan/tasks/{taskID}/worker-result`, then acknowledges the Redis message with `XACK`.
- Middleware availability can be checked with Worker CLI `python3 -m app.main --redis-ping`; it only sends Redis `PING` and does not read or acknowledge Stream messages.

Worker logs are JSON lines and include `taskId`, stage/event name, duration in milliseconds, error messages when present, and `modelVersion`.

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

## Core Data Model

The Go API schema now keeps the main business entities in relational tables so dashboard, scan, grading, and analytics queries share the same source of truth.

Organization and identity:

- `schools`, `campuses`, `grades`, `classes`
- `users`, `roles`, `user_roles`
- `teachers`, `teacher_classes`
- `students`, `guardians`, `student_guardians`

Exam and assignment lifecycle:

- `exams`: exam metadata, subject, grade, bound template id/version, status, start/end time
- `assignments`: class-facing work item with `exam_id`, `kind`, `class_id`, `template_id`, `template_version`, `teacher_id`, `published_at`, `due_at`, `completed_at`, `status`
- `assignment_classes`: published classes for an assignment
- `scan_jobs`: scan/OCR task queue state, fixed `template_version`, retry and queue fields
- `submissions`: student submission state, scan task binding, matched student/class, page count, grading time

Template and question structure:

- `paper_templates`: versioned template metadata. Only `published` templates can be bound to scan tasks.
- `question_templates`: question number, type, score, standard answer, scoring rules, knowledge points, and answer-region coordinates.
- `question_types`: objective/subjective type definitions and auto-grade capability.
- `knowledge_points`: subject knowledge taxonomy.

Grading and traceability:

- `student_answers`: normalized answer text/image per submission-question.
- `ocr_results`: OCR text, confidence, provider, block JSON, and source object key.
- `objective_grades`: objective question score, answer comparison, confidence.
- `subjective_reviews`: AI suggestion and pending teacher review queue.
- `grading_decisions`: final teacher decision per submission-question.
- `question_scores`: final per-question score used by reports and analytics.
- `grading_history`: append-only grading action history.

Wrong-question archive:

- `wrong_questions`: student, submission, question, knowledge point, score/max score, correct answer, student answer, teacher explanation, correction status, repractice status, correction time.

Object storage metadata:

- `object_files`: object key, bucket, storage driver, public URL, content type, byte size, purpose, owner type/id, metadata JSON.
- Current purposes include `template_source`, `student_answer`, and `scan_upload`; the same table is reserved for cropped answer images, OCR intermediate files, and report exports.

## Subjective Decision Persistence

`POST /api/grading/subjective/decision` writes one transaction:

- upserts `grading_decisions`
- upserts final `question_scores`
- appends `grading_history`
- upserts `student_answers` from OCR/review text
- marks `subjective_reviews.status = reviewed`
- marks the submission as `graded`
- creates or updates `wrong_questions` when `finalScore < fullScore`

Response:

```json
{
  "status": "saved",
  "finalScore": 8,
  "nextQuestion": "q_018",
  "nextReview": {
    "reviewId": "review_002",
    "submissionId": "sub_002",
    "questionId": "q_018"
  }
}
```

## Subjective Review Workbench

`GET /api/grading/subjective/reviews` returns the current teacher review queue. Web filters this client-side by exam/paper, class, question number, confidence, and status.

```json
{
  "items": [
    {
      "id": "review_001",
      "studentName": "张三",
      "className": "六年级 3 班",
      "paperName": "六年级数学期中卷",
      "questionNo": "15",
      "aiAdvice": "8 / 10",
      "confidence": 86,
      "status": "pending",
      "reviewStage": "first_review"
    }
  ]
}
```

`GET /api/grading/subjective/history?submissionId={id}&questionId={id}` returns score changes and review audit records.

```json
{
  "items": [
    {
      "id": 1,
      "submissionId": "sub_001",
      "questionId": "q_015",
      "action": "modified",
      "score": 8,
      "note": "步骤完整，结果正确，表达略不规范。",
      "actorName": "陈老师",
      "reviewStage": "first_review",
      "modelVersion": "mock-ai-worker-v1",
      "createdAt": "2026-06-20T10:20:00+08:00"
    }
  ]
}
```

Workbench interactions:

- score input is bounded to `0..fullScore` with `0.5` step validation
- quick actions support AI score, full score, zero score, and plus/minus `0.5`
- keyboard shortcuts: `A` accept AI, `M` save modified score, `R` reject AI, `F` full score, `Z` zero score, `N` next, `B` previous
- review stages: `first_review`, `second_review`, `spot_check`, `arbitration`
- decision values include `accepted_ai`, `modified`, `rejected`, `second_review`, `arbitration`, `spot_check`
- image tools support zoom, rotate, drag/pan, reset, and current-question highlight

## API Error Response

All error responses use the same envelope:

```json
{
  "error": {
    "code": "VALIDATION_REQUIRED",
    "message": "title, className and templateId are required",
    "field": "templateId"
  }
}
```

`field` is optional. Current error codes are:

- `VALIDATION_REQUIRED`
- `BAD_REQUEST`
- `FORBIDDEN`
- `NOT_FOUND`
- `CONFLICT`
- `SERVICE_UNAVAILABLE`
- `INTERNAL_ERROR`
- `REQUEST_ERROR`
