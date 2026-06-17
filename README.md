# Club - AI 阅卷与学情平台

Club 是一个面向学校、培训机构和教师的 AI 智能阅卷与学情分析系统。第一阶段聚焦可落地闭环：

```text
试卷模板 -> 扫描导入 -> OCR/OMR 阅卷 -> 教师复核 -> 成绩统计 -> 错题归档 -> 学情分析
```

## 工程结构

```text
club/
  apps/
    web/              # 教师 Web 端，现代轻量 SaaS 风
    mobile/           # React Native 移动端，Android 优先配置
  services/
    api/              # Go 主业务 API
    ai-worker/        # Python OCR/AI Worker 骨架
  packages/
    shared/           # 共享类型与接口说明
  docs/
    product.md        # 产品拆解
    prototype.md      # 原型说明
  infra/
    docker-compose.yml
```

## 首版能力

- 教师工作台：扫描队列、待复核试卷、未提交作业、薄弱知识点。
- 主观题批阅：标准答案与学生试卷左右分屏对比，AI 给出建议分和理由，教师最终确认。
- 试卷模板：题目区域、题型、分值、知识点、标准答案。
- 学情分析：试卷分析、题目正确率、知识点掌握度、学生错题分布。
- 学生/家长端：任务查看、拍照上传、错题本。

## 本地启动

Go API:

```bash
cd services/api
go run ./cmd/server
```

Python AI Worker:

```bash
cd services/ai-worker
python3 -m app.main
```

Web:

```bash
cd apps/web
npm install
npm run dev
```

Mobile Android:

```bash
cd apps/mobile
npm install
npm run android
```

