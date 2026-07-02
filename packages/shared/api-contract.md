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
- `POST /api/analytics/generate-scores`
- `GET /api/analytics/export/scores.csv`
- `GET /api/mistakes`
- `GET /api/mistakes/{mistakeID}`
- `PATCH /api/mistakes/{mistakeID}/knowledge-points`
- `POST /api/mistakes/repractice`
- `GET /api/learning/profile`
- `GET /api/reports/guardian`
- `GET /api/knowledge-points`
- `POST /api/knowledge-points`
- `GET /api/question-bank`
- `POST /api/question-bank`
- `GET /api/paper-compositions`
- `POST /api/paper-compositions`
- `POST /api/paper-compositions/{compositionID}/ai-compose-request`
- `POST /api/paper-analysis/blank-paper-uploads`
- `POST /api/answer-sheets/uploads`
- `POST /api/grading/tasks`
- `GET /api/organization/graph`
- `POST /api/organization/{kind}` (`schools|grades|subjects|classes|teachers|students`)
- `POST /api/guardian/invitations`
- `POST /api/guardian/certifications`
- `GET /api/guardian/certifications?status=pending`
- `PATCH /api/guardian/certifications/{certificationID}`
- `GET /api/portal/student?studentId={studentID}`
- `GET /api/portal/guardian?guardianId={guardianID}&studentId={studentID}`
- `POST /api/ai/capabilities/{analysis|ladder}/requests`
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

`POST /api/scan/uploads` accepts multipart form files in the `files` field. Supported formats are PDF, PNG, JPG, WebP, and ZIP scan packages. Each file is limited to 25 MB. When `storageDriver=obs`, the API uploads through the Huawei OBS SDK and stores the real bucket/key/URL metadata; other drivers currently use the local upload store.

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

OMR/OCR objective answers in Worker write-back are persisted by the Go API when `result.omrResults[].answers[]` contains `questionNo`, `selected`, and `confidence`. The API matches answers to the task's published template and submissions, writes `objective_grades` and `question_scores`, and creates `objective_review_exceptions` for low-confidence or empty answers so they do not silently become final scores.

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

## Objective Grading And Analytics

`GET /api/analytics/classroom` returns the class score overview, score bands, question-level statistics, student-level rankings, weak knowledge points, and objective-answer exceptions.

```json
{
  "className": "六年级 3 班",
  "averageScore": 81.6,
  "highestScore": 98,
  "lowestScore": 54,
  "studentCount": 42,
  "gradedCount": 40,
  "completionRate": 95,
  "passRate": 88,
  "excellentRate": 22,
  "scoreBands": [
    { "label": "80-89", "min": 80, "max": 89, "count": 18 }
  ],
  "questionDetails": [
    {
      "no": "18",
      "type": "应用题",
      "accuracy": 38,
      "scoreRate": 44,
      "difficulty": "偏难",
      "discrimination": 81,
      "typicalError": "图形拆分和公式迁移错误"
    }
  ],
  "studentScores": [
    {
      "studentName": "李四",
      "className": "六年级 3 班",
      "score": 72,
      "rank": 4,
      "weakness": ["分数应用题", "比例"]
    }
  ],
  "objectiveExceptions": [
    {
      "id": 1,
      "submissionId": "sub_002",
      "studentName": "李四",
      "questionId": "q_001",
      "questionNo": "1",
      "answer": "B",
      "confidence": 68,
      "reason": "低置信度且答案与标准答案不一致",
      "status": "pending",
      "suggestedScore": 0
    }
  ]
}
```

`POST /api/analytics/generate-scores?className=六年级%203%20班` runs a database transaction that copies objective grades into final question scores, sums all question scores per submission, upserts `exam_scores`, and marks submissions as graded.

```json
{
  "status": "generated",
  "className": "六年级 3 班",
  "generated": 40
}
```

`GET /api/analytics/export/scores.csv` downloads a UTF-8 CSV score sheet with student, class, total score, rank, and weak knowledge points.

## Wrong Questions And Learning Profile

`GET /api/mistakes` returns automatically archived objective and subjective mistakes. Optional query parameters are `paper`, `className`, `studentName`, `knowledge`, `errorType`, and `search`. Standard `errorType` values are `concept`, `calculation`, `reading`, `expression`, and `other`.

```json
{
  "items": [
    {
      "id": 2,
      "studentName": "张三",
      "className": "六年级 3 班",
      "questionNo": "15",
      "questionType": "subjective",
      "knowledgePoint": "比例",
      "errorType": "expression",
      "wrongReason": "比例关系书写不规范",
      "sourcePaper": "六年级数学期中卷",
      "originalQuestion": "根据比例关系解决实际问题。",
      "score": 8,
      "maxScore": 10,
      "correctAnswer": "设未知数并列比例求解，结果为 24 千克。",
      "studentAnswer": "3/5 = x/40，x = 24。",
      "explanation": "建模正确，补充规范比例式和单位说明。",
      "correctionStatus": "pending",
      "repracticeStatus": "not_assigned"
    }
  ]
}
```

`GET /api/mistakes/{mistakeID}` returns one full review record. `POST /api/mistakes/repractice` creates an assignment linked to the selected mistakes and knowledge points.

```json
{
  "wrongQuestionIds": [1, 2],
  "title": "错题订正与再练",
  "dueAt": "2026-06-23 18:00:00"
}
```

`PATCH /api/mistakes/{mistakeID}/knowledge-points` rebinds one wrong-question record to stable knowledge point ids. The legacy `wrong_questions.knowledge_point` text field is still updated for compatibility, while `wrong_question_knowledge_points` stores the normalized relation.

```json
{
  "knowledgePointIds": ["kp_001", "kp_004"]
}
```

## Question Bank And Knowledge Points

Phase 3 introduces an independent question bank. `question_templates` remain bound to paper templates; `question_bank` is the reusable source for future manual and AI-assisted paper composition.

`GET /api/knowledge-points?subject=数学&gradeId=grade_6&search=比例` returns normalized knowledge tags.

```json
{
  "items": [
    {
      "id": "kp_004",
      "schoolId": "school_001",
      "gradeId": "grade_6",
      "subjectId": "subject_math",
      "name": "比例",
      "subject": "数学",
      "code": "math_ratio"
    }
  ],
  "counts": {
    "knowledgePoints": 1
  }
}
```

`POST /api/knowledge-points` creates a knowledge tag.

```json
{
  "schoolId": "school_001",
  "gradeId": "grade_6",
  "subjectId": "subject_math",
  "name": "圆柱体积",
  "subject": "数学",
  "code": "math_cylinder_volume"
}
```

`GET /api/question-bank` supports `subject`, `gradeId`, `knowledge`, `type`, `difficulty`, and `search` filters. `knowledge` may be either a knowledge point id or name.

```json
{
  "items": [
    {
      "id": "qb_002",
      "schoolId": "school_001",
      "gradeId": "grade_6",
      "subjectId": "subject_math",
      "subject": "数学",
      "questionType": "subjective",
      "difficulty": "medium",
      "content": "一桶油用去 3/5 后还剩 40 千克，这桶油原来有多少千克？",
      "answer": "100 千克",
      "analysis": "剩余为 2/5，对应 40 千克，所以总量为 40 ÷ 2/5 = 100。",
      "source": "manual",
      "status": "active",
      "knowledge": [
        { "id": "kp_001", "name": "分数应用题", "subject": "数学" },
        { "id": "kp_004", "name": "比例", "subject": "数学" }
      ],
      "linkedMistakes": 1
    }
  ],
  "counts": {
    "questions": 1,
    "knowledgePoints": 5,
    "linkedMistakes": 3
  }
}
```

`POST /api/question-bank` creates a reusable question and links it to at least one knowledge point. The request can use existing `knowledgePointIds` or plain `knowledgePoints`; missing names are created automatically under the supplied subject.

```json
{
  "schoolId": "school_001",
  "gradeId": "grade_6",
  "subjectId": "subject_math",
  "subject": "数学",
  "questionType": "subjective",
  "difficulty": "medium",
  "content": "一桶油用去 3/5 后还剩 40 千克，这桶油原来有多少千克？",
  "answer": "100 千克",
  "analysis": "剩余为 2/5，对应 40 千克。",
  "source": "manual",
  "createdBy": "teacher_001",
  "knowledgePoints": ["分数应用题", "比例"]
}
```

## Paper Composition And Grading Flow

Phase 4 adds the teacher-facing paper composition and grading workflow foundation. AI-related endpoints create `ai_tasks` records with `provider=third_party_reserved` and `status=pending`. Phase 5 adds a separate third-party AI dispatch layer; creating a task still does not call a model provider until the task is explicitly dispatched.

`GET /api/paper-compositions` lists manual paper drafts and their selected question-bank items.

```json
{
  "items": [
    {
      "id": "paper_178...",
      "title": "分数与比例专项练习",
      "gradeId": "grade_6",
      "gradeName": "六年级",
      "subjectId": "subject_math",
      "subject": "数学",
      "mode": "manual",
      "status": "draft",
      "questionCount": 2,
      "totalScore": 10,
      "questions": [
        {
          "id": "qb_002",
          "content": "一桶油用去 3/5 后还剩 40 千克，这桶油原来有多少千克？",
          "sortOrder": 1,
          "score": 5
        }
      ]
    }
  ],
  "counts": { "compositions": 1 }
}
```

`POST /api/paper-compositions` saves a manual paper draft from selected question-bank ids.

```json
{
  "title": "分数与比例专项练习",
  "schoolId": "school_001",
  "gradeId": "grade_6",
  "gradeName": "六年级",
  "subjectId": "subject_math",
  "subject": "数学",
  "createdBy": "teacher_001",
  "questionIds": ["qb_001", "qb_002"],
  "scores": { "qb_001": 2, "qb_002": 8 }
}
```

`POST /api/paper-compositions/{compositionID}/ai-compose-request` creates a reserved third-party AI paper-composition task. The request can include target knowledge points, difficulty, question count, and teacher id.

```json
{
  "createdBy": "teacher_001",
  "knowledgePointIds": ["kp_001", "kp_004"],
  "difficulty": "medium",
  "phase": "reserved"
}
```

Response:

```json
{
  "id": "aitask_178...",
  "taskType": "ai_paper_composition",
  "status": "pending",
  "provider": "third_party_reserved",
  "message": "第三方 AI 任务已创建，等待人工或调度器派发"
}
```

`POST /api/paper-analysis/blank-paper-uploads` uploads one or more blank paper files and creates reserved `paper_template_analysis` tasks. Files use the same validation and OBS/local storage path as scan uploads. The endpoint stores object metadata with purpose `blank_paper`.

Multipart fields:

- `files`: PDF/PNG/JPG/WebP/ZIP
- `compositionId`: optional paper draft id
- `createdBy`: teacher id

`POST /api/answer-sheets/uploads` uploads student answer sheets, stores object metadata with purpose `student_answer`, records `answer_sheet_uploads`, and creates reserved upload/grading-related task records.

Multipart fields:

- `files`: PDF/PNG/JPG/WebP/ZIP
- `compositionId`: paper draft id
- `studentId`: optional student id
- `studentName`: student display name
- `createdBy`: teacher id

`POST /api/grading/tasks` creates a pending grading task. `mode=standard_answer` reserves standard-answer comparison; `mode=ai` reserves third-party AI grading. Neither mode performs real grading during task creation.

```json
{
  "compositionId": "paper_178...",
  "mode": "ai",
  "createdBy": "teacher_001",
  "phase": "reserved"
}
```

## Third-party AI Task Dispatch

Phase 5 uses a generic HTTP adapter for third-party AI. The platform does not hardcode a model vendor and does not self-host AI. Provider configuration can be supplied through YAML or environment variables:

```yaml
aiProvider:
  name: generic-http
  baseUrl: https://provider.example.com/tasks
  apiKey: ""
  timeoutSeconds: 30
  callbackSecret: ""
```

Equivalent environment variables:

- `AI_PROVIDER_NAME`
- `AI_PROVIDER_BASE_URL`
- `AI_PROVIDER_API_KEY`
- `AI_PROVIDER_TIMEOUT_SECONDS`
- `AI_PROVIDER_CALLBACK_SECRET`

`GET /health` exposes only safe provider metadata:

```json
{
  "config": {
    "aiProvider": {
      "name": "generic-http",
      "baseUrl": "https://provider.example.com/tasks",
      "timeoutSeconds": 30,
      "apiKeyProvided": true,
      "callbackSecretProvided": true,
      "configured": true
    }
  }
}
```

`GET /api/ai/tasks?status=pending&taskType=ai_grading` lists the latest AI task records.

```json
{
  "items": [
    {
      "id": "aitask_178...",
      "taskType": "ai_grading",
      "status": "pending",
      "provider": "third_party_reserved",
      "request": { "compositionId": "paper_178..." },
      "ownerType": "paper_composition",
      "ownerId": "paper_178...",
      "message": "第三方 AI 任务已创建，等待人工或调度器派发"
    }
  ],
  "counts": { "tasks": 1 }
}
```

`POST /api/ai/tasks/{taskID}/dispatch` sends the task to `aiProvider.baseUrl` when `baseUrl` and `apiKey` are configured. The outbound request uses `Authorization: Bearer {apiKey}` and this JSON shape:

```json
{
  "taskId": "aitask_178...",
  "taskType": "ai_grading",
  "request": { "compositionId": "paper_178..." },
  "sourceObjectKey": "uploads/...",
  "sourceUrl": "/uploads/...",
  "ownerType": "paper_composition",
  "ownerId": "paper_178...",
  "createdBy": "teacher_001",
  "callbackPath": "/api/ai/tasks/aitask_178.../callback"
}
```

Dispatch responses:

- Missing `baseUrl` or `apiKey`: task status becomes `config_required`, response status `409`.
- Provider returns 2xx: task status becomes `processing`; provider response is stored in `result_json`.
- Provider returns non-2xx or network error: task status becomes `failed`; error details are stored in `error_message`.

`POST /api/ai/tasks/{taskID}/callback` lets the third-party Provider write back progress or final results. If `callbackSecret` is configured, the Provider must send `X-AI-Callback-Secret`.

```json
{
  "status": "succeeded",
  "result": {
    "summary": "已完成阅卷",
    "scores": []
  }
}
```

Supported callback statuses: `processing`, `succeeded`, and `failed`. For compatibility, the API also accepts `completed` and stores it as `succeeded`.

`GET /api/learning/profile?className=六年级%203%20班` returns class knowledge mastery with current/previous values, trend, error count, affected-student count, student risks, and missing-work alerts. Score generation stores a daily mastery snapshot calculated from per-question score rate and wrong-question frequency, so trends can compare multiple exams.

`GET /api/reports/guardian?studentName=李四` returns a simplified guardian-facing summary with the latest score, mistake count, weak knowledge points, and concrete home-study actions.

## Student And Guardian Portal

`GET /api/portal/student?studentId={studentID}` returns the Phase 2 student learning workspace. Student and guardian portal responses are scoped to the selected learner only: personal latest score, personal homework status, personal score trend, personal mistakes, weak knowledge points, and AI product entries. Class/grade aggregate analytics, rankings, highest/lowest scores, and other-student dimensions are intentionally not returned here; those remain available only through teacher, researcher, and administrator analytics endpoints.

```json
{
  "studentId": "stu_001",
  "studentName": "张三",
  "gradeName": "六年级",
  "className": "六年级 3 班",
  "scoreSummary": {
    "gradeName": "六年级",
    "className": "六年级 3 班",
    "personal": 85
  },
  "homeworkSummary": {
    "total": 2,
    "completed": 1,
    "pending": 1,
    "overdue": 0,
    "completion": 50,
    "needsAttention": 1
  },
  "homework": [
    {
      "id": "assign_001",
      "title": "六年级数学期中卷",
      "subject": "数学",
      "status": "graded",
      "dueAt": "2026-06-23 18:00"
    }
  ],
  "scoreTrend": [
    { "label": "六年级数学期中卷", "score": 85 }
  ],
  "mistakes": [
    {
      "subject": "数学",
      "paperCount": 1,
      "homeworkCount": 0,
      "items": []
    }
  ],
  "weakPoints": ["比例"],
  "ai": [
    {
      "key": "analysis",
      "name": "AI 学情分析",
      "status": "planned",
      "description": "多维分析学科与知识点短板，输出补漏地图",
      "cta": "登记分析意向",
      "priceLabel": "后续付费"
    }
  ],
  "offers": [
    {
      "key": "analysis",
      "name": "AI 学情深度分析",
      "description": "把成绩、作业和错题整理成知识点短板与掌握程度，生成可解释的补漏地图。",
      "cta": "预约开通",
      "priceLabel": "即将开放"
    }
  ]
}
```

`GET /api/portal/guardian?guardianId={guardianID}&studentId={studentID}` returns the approved children for the guardian plus the selected child's portal data. The `student_guardians` table is the access gate; a guardian cannot request a child that has not passed certification.

```json
{
  "guardianId": "guardian_001",
  "children": [
    {
      "studentId": "stu_001",
      "studentName": "张三",
      "gradeName": "六年级",
      "className": "六年级 3 班"
    }
  ],
  "selected": {
    "studentId": "stu_001",
    "studentName": "张三"
  }
}
```

`POST /api/ai/capabilities/{analysis|ladder}/requests` records a waitlist/intent record only. It does not call a model provider in the current phase.

```json
{
  "studentId": "stu_001",
  "userId": "guardian_001",
  "channel": "guardian"
}
```

## Core Data Model

The Go API schema now keeps the main business entities in relational tables so dashboard, scan, grading, and analytics queries share the same source of truth.

Organization and identity:

- `schools`, `campuses`, `grades`, `classes`
- `users`, `roles`, `user_roles`
- `teachers`, `teacher_classes`, `subjects`, `class_subjects`, `teacher_grades`, `teacher_subjects`
- `students`, `guardians`, `student_guardians`
- `guardian_invitations`, `guardian_certifications`

## Organization And Guardian Certification

`GET /api/organization/graph` returns the school tree, aggregate counts, selectable organization lists, and pending guardian certification records. The selectable lists are used by the Web admin screen to create real relations instead of relying on hard-coded ids.

```json
{
  "counts": {
    "schools": 1,
    "grades": 1,
    "classes": 1,
    "teachers": 1,
    "students": 3,
    "subjects": 3,
    "classSubjects": 1,
    "pendingCertifications": 0
  },
  "schools": [
    {
      "id": "school_001",
      "name": "示范学校",
      "type": "school",
      "children": [
        {
          "id": "grade_6",
          "name": "六年级",
          "type": "grade",
          "children": [
            { "id": "class_603", "name": "六年级 3 班", "type": "class" }
          ]
        }
      ]
    }
  ],
  "grades": [{ "id": "grade_6", "name": "六年级", "schoolId": "school_001" }],
  "classes": [{ "id": "class_603", "name": "六年级 3 班", "schoolId": "school_001", "gradeId": "grade_6" }],
  "subjects": [{ "id": "subject_math", "name": "数学", "schoolId": "school_001" }],
  "classSubjects": [{ "id": "course_603_math", "name": "六年级 3 班 · 数学", "schoolId": "school_001", "gradeId": "grade_6", "classId": "class_603", "subjectId": "subject_math", "teacherId": "teacher_001", "meta": "陈老师" }],
  "teachers": [{ "id": "teacher_001", "name": "陈老师", "schoolId": "school_001", "gradeIds": ["grade_6"], "subjectIds": ["subject_math"], "meta": "数学" }],
  "students": [{ "id": "stu_001", "name": "张三", "schoolId": "school_001", "gradeId": "grade_6", "classId": "class_603", "studentNo": "60301" }],
  "certifications": []
}
```

`POST /api/organization/{kind}` creates Phase 1 entities and their required relations.

- `schools`: `{ "name": "第一实验学校" }`
- `grades`: `{ "name": "七年级", "schoolId": "school_001", "stage": "middle" }`
- `subjects`: `{ "name": "物理", "schoolId": "school_001", "code": "physics" }`
- `classes`: `{ "name": "七年级 1 班", "schoolId": "school_001", "gradeId": "grade_7" }`
- `class-subjects`: `{ "schoolId": "school_001", "gradeId": "grade_7", "classId": "class_701", "subjectId": "subject_physics", "teacherId": "teacher_002" }`
- `teachers`: `{ "name": "王老师", "schoolId": "school_001", "gradeId": "grade_7", "subjectId": "subject_physics", "classId": "class_701", "mobile": "13800000000" }`
- `students`: `{ "name": "小明", "classId": "class_701", "studentNo": "70101" }`

Class-subject creation writes `class_subjects` and is the curriculum-level relation that defines which subjects a class offers. `teacherId` is optional and can assign the current course teacher. Teacher creation still writes `users`, `teachers`, `teacher_grades`, `teacher_subjects`, and optionally `teacher_classes`. Student creation writes `users` and `students`.

`POST /api/guardian/invitations` creates a teacher-scoped guardian invite. The teacher must be assigned to the student's grade before the invite can be created.

```json
{
  "teacherId": "teacher_001",
  "studentId": "stu_001",
  "mobileHint": "13800000011"
}
```

Response includes the one-time token and an invite path:

```json
{
  "id": "invite_...",
  "token": "a1b2...",
  "invitePath": "/guardian/certify?token=a1b2...",
  "expiresAt": "2026-07-02T12:00:00+08:00"
}
```

`POST /api/guardian/certifications` submits a guardian certification request. The requester can use an existing `guardianId`, or provide `guardianName` and `mobile` to create a guardian account before review.

```json
{
  "token": "a1b2...",
  "guardianName": "张三家长",
  "mobile": "13800000011",
  "relationship": "parent"
}
```

The certification stays `pending`; `student_guardians` is not written at submit time.

`PATCH /api/guardian/certifications/{certificationID}` approves or rejects the request.

```json
{
  "status": "approved",
  "reviewerId": "user_admin_001",
  "reviewNote": "信息匹配，允许访问"
}
```

Only approved requests write or update `student_guardians`, which is the access gate for guardian portal data and multi-child switching.

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
- `question_bank`: reusable questions independent from paper templates.
- `question_bank_knowledge_points`: normalized question-to-knowledge relations for search and future paper composition.
- `paper_compositions`: teacher-created paper drafts.
- `paper_composition_questions`: selected question-bank items, order, and score inside a paper draft.
- `ai_tasks`: third-party AI task queue for paper analysis, AI composition, AI grading, student analysis, dispatch status, provider response, and callback result tracking.

Grading and traceability:

- `student_answers`: normalized answer text/image per submission-question.
- `ocr_results`: OCR text, confidence, provider, block JSON, and source object key.
- `objective_grades`: objective question score, answer comparison, confidence.
- `objective_review_exceptions`: low-confidence, missing, or abnormal objective answers waiting for manual confirmation.
- `subjective_reviews`: AI suggestion and pending teacher review queue.
- `grading_decisions`: final teacher decision per submission-question.
- `question_scores`: final per-question score used by reports and analytics.
- `grading_history`: append-only grading action history.

Wrong-question archive:

- `wrong_questions`: student, submission, question, primary knowledge point, score/max score, correct answer, student answer, teacher explanation, correction status, repractice status, correction time.
- `wrong_question_knowledge_points`: normalized wrong-question-to-knowledge relations used by student/guardian weak-point views, future AI analysis, and future targeted practice generation.
- `repractice_tasks`: selected wrong-question ids, linked knowledge points, due time, and assignment status.
- `knowledge_mastery_history`: dated class/student mastery snapshots used for multi-exam trend comparison.

Object storage metadata:

- `object_files`: object key, bucket, storage driver, public URL, content type, byte size, purpose, owner type/id, metadata JSON.
- `answer_sheet_uploads`: student answer sheet files linked to paper drafts and reserved grading tasks.
- Current object file purposes include `template_source`, `blank_paper`, `student_answer`, and `scan_upload`; the same table is reserved for cropped answer images, OCR intermediate files, and report exports.

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
