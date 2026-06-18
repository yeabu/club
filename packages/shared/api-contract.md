# API Contract Draft

## Endpoints

- `GET /health`
- `GET /api/dashboard`
- `GET /api/grading/subjective/current`
- `GET /api/grading/subjective/reviews/{reviewID}`
- `POST /api/grading/subjective/decision`
- `GET /api/templates`
- `GET /api/analytics/classroom`
- `GET /api/dev/connections`
- `POST /api/dev/reset-demo`

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
