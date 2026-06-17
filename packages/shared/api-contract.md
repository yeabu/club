# API Contract Draft

## Endpoints

- `GET /health`
- `GET /api/dashboard`
- `GET /api/grading/subjective/current`
- `POST /api/grading/subjective/decision`
- `GET /api/templates`
- `GET /api/analytics/classroom`

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

