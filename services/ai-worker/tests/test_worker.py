from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from app.main import Worker, WorkerConfig, analyze_paper, detect_wrong_reason, grade_subjective, parse_xread_response


class WorkerPipelineTest(unittest.TestCase):
    def config(self, result_dir: Path) -> WorkerConfig:
        return WorkerConfig(
            app_env="test",
            port=8090,
            redis_addr="127.0.0.1:6379",
            redis_db=0,
            redis_password="",
            redis_stream="club:scan:tasks",
            redis_group="club-ai-worker",
            redis_consumer="test-worker",
            api_base_url="",
            result_dir=result_dir,
            sample_manifest=Path("samples/manifest.json"),
            model_version="test-model",
            storage_driver="local",
            enable_redis_consumer=False,
        )

    def test_template_analysis_returns_reviewable_regions(self) -> None:
        result = analyze_paper({"paperName": "测试卷"})
        self.assertEqual(result["paperName"], "测试卷")
        self.assertGreaterEqual(len(result["suggestedQuestions"]), 2)
        self.assertIn("region", result["suggestedQuestions"][0])

    def test_subjective_grading_has_score_reason_and_evidence(self) -> None:
        result = grade_subjective(
            {
                "fullScore": 10,
                "standardAnswer": "x = 24",
                "studentAnswer": "设 x，x = 24，答需要 24 千克。",
                "scoringRules": ["列式 4 分", "计算 2 分"],
            }
        )
        self.assertGreater(result["score"], 0)
        self.assertGreater(result["confidence"], 70)
        self.assertTrue(result["evidence"])

    def test_wrong_reason_classifies_incomplete_expression(self) -> None:
        result = detect_wrong_reason({"studentAnswer": "x = 24", "knowledge": ["比例"]})
        self.assertEqual(result["errorType"], "表达不完整")
        self.assertIn("trainingHint", result)

    def test_process_scan_task_writes_local_result(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            worker = Worker(self.config(Path(temp_dir)))
            result = worker.process_scan_task(
                {
                    "taskId": "scan_test_001",
                    "title": "测试扫描",
                    "templateId": "tpl_001",
                    "templateVersion": 1,
                    "fileKeys": ["samples/student-paper.sample.json"],
                },
                callback=False,
            )
            self.assertEqual(result["status"], "识别完成")
            self.assertTrue(Path(result["localResultPath"]).exists())
            self.assertTrue(result["ocrResults"])
            self.assertTrue(result["omrResults"])

    def test_parse_redis_stream_response(self) -> None:
        response = [["club:scan:tasks", [["1-0", ["taskId", "scan_001", "payload", "{\"taskId\":\"scan_001\"}"]]]]]
        entries = parse_xread_response(response)
        self.assertEqual(entries[0][0], "1-0")
        self.assertEqual(entries[0][1]["taskId"], "scan_001")


if __name__ == "__main__":
    unittest.main()
