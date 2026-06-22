import {
  AlertCircle,
  Bell,
  BookOpenCheck,
  Check,
  ClipboardCheck,
  Cloud,
  Database,
  FileStack,
  LayoutDashboard,
  Loader2,
  MessageSquareText,
  Move,
  PenLine,
  RefreshCw,
  RotateCcw,
  ScanLine,
  Send,
  SlidersHorizontal,
  Sparkles,
  StepForward,
  TimerReset,
  ZoomIn,
  ZoomOut,
  ShieldCheck,
  UsersRound
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

type Metric = {
  label: string;
  value: string;
  delta: string;
  tone: "primary" | "warning" | "danger" | "success";
};

type ScanJob = {
  id: string;
  title: string;
  className: string;
  templateId?: string;
  templateVersion?: number;
  pages: number;
  notes?: string;
  status: string;
  progress: number;
  failureReason?: string;
  retryCount: number;
  queueStatus?: string;
  queueMessage?: string;
  files?: ScanUploadFile[];
};

type ScanUploadFile = {
  key: string;
  fileName: string;
  contentType: string;
  size: number;
  url: string;
  page?: number;
  status?: string;
  failureReason?: string;
  studentId?: string;
  studentName?: string;
  matchStatus?: string;
  matchMethod?: string;
};

type ScanUploadResponse = {
  files: ScanUploadFile[];
};

type ScanTaskResponse = {
  status: string;
  queueId?: string;
  queueError?: string;
  task: ScanJob;
};

type ScanTaskListResponse = {
  tasks: ScanJob[];
};

type TemplateAISuggestionResponse = {
  paperName: string;
  questionCount: number;
  totalScore: number;
  suggestedQuestions: QuestionTemplate[];
  reviewRequired: boolean;
  source: string;
};

type ScanTaskPreviewResponse = {
  task: ScanJob;
  files: ScanUploadFile[];
};

type ReviewItem = {
  id: string;
  studentName: string;
  className?: string;
  paperName: string;
  questionNo: string;
  aiAdvice: string;
  confidence: number;
  status?: string;
  reviewStage?: string;
};

type ReviewQueueResponse = {
  items: ReviewItem[];
};

type KnowledgeStat = {
  name: string;
  accuracy: number;
  wrongCount: number;
};

type WrongQuestion = {
  id: number;
  studentId: string;
  studentName: string;
  className: string;
  submissionId: string;
  questionId: string;
  questionNo: string;
  questionType: string;
  knowledgePoint: string;
  errorType: string;
  wrongReason: string;
  sourcePaper: string;
  originalQuestion: string;
  score: number;
  maxScore: number;
  correctAnswer: string;
  studentAnswer: string;
  answerImageUrl: string;
  explanation: string;
  correctionStatus: string;
  repracticeStatus: string;
  createdAt: string;
  knowledge: string[];
};

type WrongQuestionListResponse = { items: WrongQuestion[] };

type KnowledgeMastery = {
  name: string;
  mastery: number;
  previousMastery: number;
  trend: number;
  wrongCount: number;
  studentCount: number;
};

type LearningProfile = {
  className: string;
  knowledgeMastery: KnowledgeMastery[];
  studentRisks: StudentRisk[];
  homeworkWatch: HomeworkWatch[];
};

type GuardianReport = {
  studentName: string;
  className: string;
  summary: string;
  score: number;
  wrongCount: number;
  weakness: string[];
  actions: string[];
};

type PortalData = {
  studentId: string;
  studentName: string;
  gradeName: string;
  className: string;
  scoreSummary: { highest: number; lowest: number; average: number; personal: number };
  homework: Array<{ id: string; title: string; subject: string; status: string; dueAt: string }>;
  scoreTrend: Array<{ label: string; score: number }>;
  mistakes: Array<{ subject: string; paperCount: number; homeworkCount: number; items: WrongQuestion[] }>;
  ai: Array<{ key: string; name: string; status: string; description: string }>;
};

type OrganizationGraph = {
  counts: Record<string, number>;
  schools: Array<{ id: string; name: string; type: string; children?: Array<{ id: string; name: string; type: string; children?: Array<{ id: string; name: string; type: string }> }> }>;
};

type HomeworkWatch = {
  studentName: string;
  className: string;
  missing: number;
  guardian: string;
};

type DashboardData = {
  source?: "database" | "fixtures" | "local";
  metrics: Metric[];
  scanQueue: ScanJob[];
  reviewQueue: ReviewItem[];
  weakPoints: KnowledgeStat[];
  homeworkWatch: HomeworkWatch[];
};

type SubjectiveData = {
  reviewId: string;
  submissionId: string;
  questionId: string;
  paperName: string;
  studentName: string;
  className: string;
  questionNo: string;
  fullScore: number;
  standardAnswer: {
    content: string;
    scoringRules: string[];
    knowledge: string[];
  };
  studentAnswer: {
    ocrText: string;
    imageUrl: string;
  };
  ai: {
    score: number;
    reason: string;
    comments: string[];
    confidence: number;
    modelVersion?: string;
  };
};

type GradingHistoryItem = {
  id: number;
  submissionId: string;
  questionId: string;
  action: string;
  score: number;
  note: string;
  actorName: string;
  reviewStage: string;
  modelVersion: string;
  createdAt: string;
};

type GradingHistoryResponse = {
  items: GradingHistoryItem[];
};

type GradingDecisionResponse = {
  status: string;
  finalScore: number;
  nextQuestion: string;
  nextReview?: SubjectiveData;
};

type TemplateMutationResponse = {
  status: string;
  template: PaperTemplate;
};

type TemplateRegionMutationResponse = {
  status: string;
  question: QuestionTemplate;
  template: PaperTemplate;
};

type Region = {
  page: number;
  x: number;
  y: number;
  width: number;
  height: number;
};

type QuestionTemplate = {
  id: string;
  no: string;
  type: string;
  score: number;
  standardAnswer?: string;
  scoringRules?: string[];
  knowledge: string[];
  region: Region;
};

type PaperTemplate = {
  id: string;
  name: string;
  subject: string;
  grade: string;
  questionCount: number;
  totalScore: number;
  sourceFileUrl?: string;
  status?: TemplateStatus | "ready";
  version?: number;
  parentId?: string;
  questions: QuestionTemplate[];
};

type QuestionStat = {
  no: string;
  accuracy: number;
  type: string;
};

type ScoreBand = {
  label: string;
  min: number;
  max: number;
  count: number;
};

type QuestionDetailStat = {
  no: string;
  type: string;
  accuracy: number;
  scoreRate: number;
  difficulty: string;
  discrimination: number;
  typicalError: string;
};

type StudentRisk = {
  studentName: string;
  risk: string;
  weakness: string[];
};

type StudentScoreSummary = {
  studentName: string;
  className: string;
  score: number;
  rank: number;
  weakness: string[];
};

type ObjectiveReviewException = {
  id: number;
  submissionId: string;
  studentName: string;
  questionId: string;
  questionNo: string;
  answer: string;
  confidence: number;
  reason: string;
  status: string;
  suggestedScore: number;
};

type ClassroomAnalytics = {
  className: string;
  averageScore: number;
  highestScore: number;
  lowestScore: number;
  studentCount: number;
  gradedCount: number;
  completionRate: number;
  passRate: number;
  excellentRate: number;
  questionStats: QuestionStat[];
  questionDetails: QuestionDetailStat[];
  knowledgeStats: KnowledgeStat[];
  studentRisks: StudentRisk[];
  studentScores: StudentScoreSummary[];
  scoreBands: ScoreBand[];
  objectiveExceptions: ObjectiveReviewException[];
};

type ScoreGenerationResponse = {
  status: string;
  className: string;
  generated: number;
};

type ActiveView = "workspace" | "organization" | "scan" | "templates" | "grading" | "mistakes" | "analytics";
type Overlay = "filter" | "notifications" | null;
type TemplateTool = "objective" | "subjective" | "choice" | "judge";
type TemplateStatus = "draft" | "published" | "disabled";
type RequestStatus = "loading" | "processing" | "success" | "error" | "empty";
type ConnectionStatus = "checking" | "available" | "unavailable" | "skipped";
type UserRole = "teacher" | "researcher" | "admin" | "student" | "guardian";
type Permission =
  | "scan:create"
  | "template:edit"
  | "template:delete"
  | "template:ai"
  | "grading:review"
  | "grading:decide"
  | "mistake:generate"
  | "guardian:remind";

type Option = {
  label: string;
  value: string;
};

type RequestState = {
  status: RequestStatus;
  message: string;
  detail?: string;
};

type CanvasRegion = {
  id: string;
  no: string;
  type: TemplateTool;
  label: string;
  color: string;
  borderStyle: "solid" | "dashed" | "dotted";
  score: number;
  standardAnswer: string;
  scoringRules: string[];
  knowledge: string[];
  region: Region;
};

type CanvasSize = {
  label: string;
  width: number;
  height: number;
};

type TemplateSourceMode = "scan" | "library";

type TemplatePaperSource = {
  id: string;
  title: string;
  className: string;
  pages: number;
  size: CanvasSize;
  importedAt: string;
  source: "现场扫描" | "库存";
};

type TemplateDraft = {
  id: string;
  title: string;
  sourceTitle: string;
  updatedAt: string;
  size: CanvasSize;
  zoom: number;
  regions: CanvasRegion[];
};

type DragState = {
  id: string;
  mode: "move" | "resize";
  startX: number;
  startY: number;
  original: Region;
};

const templateTools: Record<TemplateTool, { label: string; color: string }> = {
  objective: { label: "客观题", color: "#155b92" },
  subjective: { label: "主观题", color: "#0d7c66" },
  choice: { label: "选择题", color: "#7c3aed" },
  judge: { label: "判断题", color: "#b97809" }
};

const objectiveAnswerOptions: Record<Exclude<TemplateTool, "subjective">, string[]> = {
  choice: ["A", "B", "C", "D"],
  judge: ["正确", "错误"],
  objective: ["A", "B", "C", "D", "正确", "错误"]
};

const templateStatusLabels: Record<TemplateStatus, string> = {
  draft: "草稿",
  published: "已发布",
  disabled: "停用"
};

const canvasPresets: CanvasSize[] = [
  { label: "A4 空白卷", width: 760, height: 1080 },
  { label: "答题卡", width: 760, height: 900 },
  { label: "横向试卷", width: 1080, height: 760 }
];

const roleConfig: Record<UserRole, { label: string; description: string; views: ActiveView[]; permissions: Permission[] }> = {
  teacher: {
    label: "教师",
    description: "批阅、导入、模板和班级学情",
    views: ["workspace", "scan", "templates", "grading", "mistakes", "analytics"],
    permissions: ["scan:create", "template:edit", "template:delete", "template:ai", "grading:review", "grading:decide", "mistake:generate", "guardian:remind"]
  },
  researcher: {
    label: "教研",
    description: "模板、错题和学情分析",
    views: ["workspace", "templates", "mistakes", "analytics"],
    permissions: ["template:edit", "template:ai", "mistake:generate"]
  },
  admin: {
    label: "管理员",
    description: "全部入口和维护操作",
    views: ["workspace", "organization", "scan", "templates", "grading", "mistakes", "analytics"],
    permissions: ["scan:create", "template:edit", "template:delete", "template:ai", "grading:review", "grading:decide", "mistake:generate", "guardian:remind"]
  },
  student: {
    label: "学生",
    description: "成绩、错题和个人学情",
    views: ["workspace"],
    permissions: []
  },
  guardian: {
    label: "家长",
    description: "完成情况、错题和薄弱点",
    views: ["workspace"],
    permissions: []
  }
};

const navItems = [
  { view: "workspace", label: "工作台", icon: LayoutDashboard },
  { view: "organization", label: "组织与用户", icon: UsersRound },
  { view: "scan", label: "扫描导入", icon: ScanLine },
  { view: "templates", label: "试卷模板", icon: FileStack },
  { view: "grading", label: "阅卷中心", icon: ClipboardCheck },
  { view: "mistakes", label: "错题集", icon: BookOpenCheck },
  { view: "analytics", label: "学情分析", icon: UsersRound }
] satisfies Array<{ view: ActiveView; label: string; icon: typeof LayoutDashboard }>;

const pageSize = 4;

const fallbackPaperSources: TemplatePaperSource[] = [
  {
    id: "paper_stock_001",
    title: "六年级数学期中卷空白卷",
    className: "六年级 3 班",
    pages: 2,
    size: canvasPresets[1],
    importedAt: "2026-06-18 09:20",
    source: "库存"
  },
  {
    id: "paper_stock_002",
    title: "分数应用题专项空白卷",
    className: "六年级 1 班",
    pages: 1,
    size: canvasPresets[0],
    importedAt: "2026-06-17 15:42",
    source: "库存"
  },
  {
    id: "paper_stock_003",
    title: "几何面积横版练习",
    className: "五年级 2 班",
    pages: 1,
    size: canvasPresets[2],
    importedAt: "2026-06-16 11:08",
    source: "库存"
  }
];

const templateDraftStorageKey = "club.templateDrafts";
const templateLibraryStorageKey = "club.templateLibrary";
const maxScanFileSizeBytes = 25 * 1024 * 1024;
const maxScanFileCount = 20;
const allowedScanExtensions = [".pdf", ".png", ".jpg", ".jpeg", ".webp", ".zip"];
const allowedScanMimeTypes = ["application/pdf", "image/png", "image/jpeg", "image/webp", "application/zip", "application/x-zip-compressed"];

const fallbackDashboard: DashboardData = {
  source: "local",
  metrics: [
    { label: "待批试卷", value: "128", delta: "较昨日 +24", tone: "primary" },
    { label: "主观题待复核", value: "36", delta: "AI 已预评分", tone: "warning" },
    { label: "未提交作业", value: "8", delta: "3 人连续未交", tone: "danger" },
    { label: "班级平均分", value: "81.6", delta: "较上次 +3.2", tone: "success" }
  ],
  scanQueue: [
    { id: "scan_001", title: "六年级数学期中卷", className: "六年级 3 班", templateId: "tpl_001", templateVersion: 1, pages: 96, status: "OCR 识别中", progress: 68, retryCount: 0, queueStatus: "queued", files: [
      { key: "mock/scan_001/zhangsan.png", fileName: "张三-第1页.png", contentType: "image/png", size: 204800, url: "/mock/student-answer-q15.png", page: 1, status: "识别中", studentId: "stu_001", studentName: "张三", matchStatus: "matched", matchMethod: "name" }
    ] },
    { id: "scan_002", title: "分数应用题专项", className: "六年级 1 班", templateId: "tpl_001", templateVersion: 1, pages: 42, status: "等待 OMR", progress: 32, retryCount: 0, queueStatus: "queued", files: [
      { key: "mock/scan_002/unmatched.png", fileName: "未匹配-第1页.png", contentType: "image/png", size: 178200, url: "/mock/student-answer-q18.png", page: 1, status: "等待 OMR", matchStatus: "pending" }
    ] },
    { id: "scan_003", title: "几何面积小测", className: "五年级 2 班", templateId: "tpl_001", templateVersion: 1, pages: 48, status: "待导入", progress: 0, failureReason: "OCR Worker 暂未消费", retryCount: 1, queueStatus: "failed", files: [
      { key: "mock/scan_003/error.png", fileName: "赵六-第1页.png", contentType: "image/png", size: 192000, url: "/mock/student-answer-q18.png", page: 1, status: "失败", failureReason: "识别超时", matchStatus: "pending" }
    ] }
  ],
  reviewQueue: [
    { id: "review_001", studentName: "张三", className: "六年级 3 班", paperName: "六年级数学期中卷", questionNo: "15", aiAdvice: "8 / 10", confidence: 86, status: "pending", reviewStage: "first_review" },
    { id: "review_002", studentName: "李四", className: "六年级 3 班", paperName: "六年级数学期中卷", questionNo: "18", aiAdvice: "6 / 8", confidence: 78, status: "second_review", reviewStage: "second_review" },
    { id: "review_003", studentName: "王五", className: "六年级 1 班", paperName: "分数应用题专项", questionNo: "7", aiAdvice: "4 / 6", confidence: 74, status: "pending", reviewStage: "first_review" }
  ],
  weakPoints: [
    { name: "分数应用题", accuracy: 42, wrongCount: 29 },
    { name: "几何面积", accuracy: 51, wrongCount: 21 },
    { name: "比例换算", accuracy: 64, wrongCount: 15 }
  ],
  homeworkWatch: [
    { studentName: "李四", className: "六年级 3 班", missing: 3, guardian: "李四家长" },
    { studentName: "赵六", className: "六年级 3 班", missing: 2, guardian: "赵六家长" }
  ]
};

const fallbackSubjective: SubjectiveData = {
  reviewId: "review_001",
  submissionId: "sub_001",
  questionId: "q_015",
  paperName: "六年级数学期中卷",
  studentName: "张三",
  className: "六年级 3 班",
  questionNo: "15",
  fullScore: 10,
  standardAnswer: {
    content: "先设未知数 x，列出比例关系 3:5 = x:40，解得 x = 24。答：需要 24 千克。",
    scoringRules: ["正确设未知数 2 分", "列出比例关系 4 分", "计算过程正确 2 分", "结果与答语完整 2 分"],
    knowledge: ["比例", "应用题建模", "方程求解"]
  },
  studentAnswer: {
    ocrText: "设需要 x 千克，3/5 = x/40，5x = 120，x = 24。答需要 24 千克。",
    imageUrl: "/mock/student-answer-q15.png"
  },
  ai: {
    score: 8,
    reason: "建模和计算结果正确，但比例式书写不够规范，缺少单位换算说明。",
    comments: ["核心步骤完整", "建议扣除书写规范 1 分", "答语完整，可保留 1 分"],
    confidence: 86
  }
};

const fallbackTemplates: PaperTemplate[] = [
  {
    id: "tpl_001",
    name: "六年级数学期中卷",
    subject: "数学",
    grade: "六年级",
    questionCount: 25,
    totalScore: 100,
    sourceFileUrl: "/mock/templates/tpl_001-blank-paper.pdf",
    status: "published",
    version: 1,
    questions: [
      { id: "q_001", no: "1", type: "single_choice", score: 2, standardAnswer: "A", scoringRules: ["选对 A 得 2 分"], knowledge: ["分数"], region: { page: 1, x: 120, y: 260, width: 480, height: 80 } },
      { id: "q_015", no: "15", type: "subjective", score: 10, standardAnswer: "先设未知数 x，列出比例关系 3:5 = x:40，解得 x = 24。", scoringRules: ["正确设未知数 2 分", "列出比例关系 4 分", "计算过程正确 2 分", "结果与答语完整 2 分"], knowledge: ["比例", "应用题建模"], region: { page: 2, x: 96, y: 420, width: 620, height: 180 } }
    ]
  },
  {
    id: "tpl_draft_001",
    name: "六年级数学期中卷 v2",
    subject: "数学",
    grade: "六年级",
    questionCount: 2,
    totalScore: 12,
    sourceFileUrl: "/mock/templates/tpl_001-blank-paper.pdf",
    status: "draft",
    version: 2,
    parentId: "tpl_001",
    questions: [
      { id: "q_draft_001", no: "1", type: "single_choice", score: 2, standardAnswer: "A", scoringRules: ["选对得分"], knowledge: ["分数"], region: { page: 1, x: 130, y: 260, width: 460, height: 80 } },
      { id: "q_draft_015", no: "15", type: "subjective", score: 10, standardAnswer: "按比例关系列式求解。", scoringRules: ["列式 4 分", "计算 4 分", "答语 2 分"], knowledge: ["比例"], region: { page: 2, x: 96, y: 420, width: 620, height: 180 } }
    ]
  },
  {
    id: "tpl_disabled_001",
    name: "旧版几何面积小测",
    subject: "数学",
    grade: "五年级",
    questionCount: 1,
    totalScore: 8,
    sourceFileUrl: "/mock/templates/old-geometry-quiz.pdf",
    status: "disabled",
    version: 1,
    questions: [
      { id: "q_disabled_018", no: "18", type: "subjective", score: 8, standardAnswer: "拆分图形后计算面积。", scoringRules: ["拆分图形 2 分", "公式正确 2 分", "计算正确 4 分"], knowledge: ["几何面积"], region: { page: 1, x: 110, y: 640, width: 600, height: 160 } }
    ]
  }
];

const fallbackAnalytics: ClassroomAnalytics = {
  className: "六年级 3 班",
  averageScore: 81.6,
  highestScore: 98,
  lowestScore: 54,
  studentCount: 42,
  gradedCount: 40,
  completionRate: 95,
  passRate: 88,
  excellentRate: 22,
  questionStats: [
    { no: "1", accuracy: 96, type: "单选题" },
    { no: "8", accuracy: 82, type: "填空题" },
    { no: "15", accuracy: 42, type: "应用题" },
    { no: "18", accuracy: 38, type: "应用题" }
  ],
  questionDetails: [
    { no: "1", type: "单选题", accuracy: 96, scoreRate: 96, difficulty: "容易", discrimination: 52, typicalError: "整体掌握较好，关注个别粗心" },
    { no: "8", type: "填空题", accuracy: 82, scoreRate: 82, difficulty: "容易", discrimination: 56, typicalError: "单位换算漏写" },
    { no: "15", type: "应用题", accuracy: 42, scoreRate: 48, difficulty: "偏难", discrimination: 78, typicalError: "比例关系建模不稳定" },
    { no: "18", type: "应用题", accuracy: 38, scoreRate: 44, difficulty: "偏难", discrimination: 81, typicalError: "图形拆分和公式迁移错误" }
  ],
  knowledgeStats: fallbackDashboard.weakPoints,
  studentRisks: [
    { studentName: "李四", risk: "连续 3 次未提交作业", weakness: ["分数应用题", "比例"] },
    { studentName: "赵六", risk: "本次低于班均 18 分", weakness: ["几何面积"] }
  ],
  studentScores: [
    { studentName: "赵六", className: "六年级 3 班", score: 88, rank: 1, weakness: ["几何面积"] },
    { studentName: "张三", className: "六年级 3 班", score: 85, rank: 2, weakness: ["表达规范"] },
    { studentName: "王五", className: "六年级 3 班", score: 82, rank: 3, weakness: ["计算基础"] },
    { studentName: "李四", className: "六年级 3 班", score: 72, rank: 4, weakness: ["分数应用题", "比例"] }
  ],
  scoreBands: [
    { label: "0-59", min: 0, max: 59, count: 1 },
    { label: "60-69", min: 60, max: 69, count: 3 },
    { label: "70-79", min: 70, max: 79, count: 10 },
    { label: "80-89", min: 80, max: 89, count: 18 },
    { label: "90-100", min: 90, max: 100, count: 8 }
  ],
  objectiveExceptions: [
    { id: 1, submissionId: "sub_002", studentName: "李四", questionId: "q_001", questionNo: "1", answer: "B", confidence: 68, reason: "低置信度且答案与标准答案不一致", status: "pending", suggestedScore: 0 }
  ]
};

const fallbackWrongQuestions: WrongQuestion[] = [
  { id: 1, studentId: "stu_002", studentName: "李四", className: "六年级 3 班", submissionId: "sub_002", questionId: "q_001", questionNo: "1", questionType: "单选题", knowledgePoint: "分数", errorType: "concept", wrongReason: "标准答案为 A，学生选择 B", sourcePaper: "六年级数学期中卷", originalQuestion: "比较两个分数的大小，选择正确答案。", score: 0, maxScore: 2, correctAnswer: "A", studentAnswer: "B", answerImageUrl: "/mock/student-answer-q18.png", explanation: "回顾分数大小比较方法。", correctionStatus: "pending", repracticeStatus: "not_assigned", createdAt: "2026-06-20T09:00:00+08:00", knowledge: ["分数"] },
  { id: 2, studentId: "stu_001", studentName: "张三", className: "六年级 3 班", submissionId: "sub_001", questionId: "q_015", questionNo: "15", questionType: "应用题", knowledgePoint: "比例", errorType: "expression", wrongReason: "比例关系书写不规范", sourcePaper: "六年级数学期中卷", originalQuestion: "根据比例关系解决实际问题。", score: 8, maxScore: 10, correctAnswer: "设未知数并列比例求解，结果为 24 千克。", studentAnswer: "3/5 = x/40，x = 24。", answerImageUrl: "/mock/student-answer-q15.png", explanation: "建模正确，补充规范比例式和单位说明。", correctionStatus: "pending", repracticeStatus: "not_assigned", createdAt: "2026-06-20T09:10:00+08:00", knowledge: ["比例"] },
  { id: 3, studentId: "stu_002", studentName: "李四", className: "六年级 3 班", submissionId: "sub_002", questionId: "q_018", questionNo: "18", questionType: "应用题", knowledgePoint: "几何面积", errorType: "calculation", wrongReason: "图形拆分后面积计算不完整", sourcePaper: "六年级数学期中卷", originalQuestion: "将组合图形拆分后计算总面积。", score: 6, maxScore: 8, correctAnswer: "长方形与三角形面积相加。", studentAnswer: "长方形面积 36，三角形面积 12。", answerImageUrl: "/mock/student-answer-q18.png", explanation: "标出拆分依据并写完整单位。", correctionStatus: "pending", repracticeStatus: "not_assigned", createdAt: "2026-06-20T09:20:00+08:00", knowledge: ["几何面积"] }
];

const fallbackLearningProfile: LearningProfile = {
  className: "六年级 3 班",
  knowledgeMastery: [
    { name: "分数应用题", mastery: 42, previousMastery: 38, trend: 4, wrongCount: 29, studentCount: 16 },
    { name: "几何面积", mastery: 51, previousMastery: 57, trend: -6, wrongCount: 21, studentCount: 13 },
    { name: "比例换算", mastery: 64, previousMastery: 59, trend: 5, wrongCount: 15, studentCount: 9 }
  ],
  studentRisks: fallbackAnalytics.studentRisks,
  homeworkWatch: fallbackDashboard.homeworkWatch
};

const fallbackGuardianReport: GuardianReport = {
  studentName: "李四",
  className: "六年级 3 班",
  summary: "本次成绩 72 分，共有 2 道题需要继续巩固。",
  score: 72,
  wrongCount: 2,
  weakness: ["分数", "几何面积"],
  actions: ["每天安排 15 分钟订正", "优先复习薄弱知识点", "完成再练后和孩子一起检查步骤"]
};

const errorTypeLabels: Record<string, string> = {
  concept: "概念错误",
  calculation: "计算错误",
  reading: "审题错误",
  expression: "表达不完整",
  other: "其他"
};

function toolFromQuestionType(type: string): TemplateTool {
  if (type === "subjective") {
    return "subjective";
  }
  if (type === "judge") {
    return "judge";
  }
  if (type === "single_choice" || type === "multiple_choice") {
    return "choice";
  }
  return "objective";
}

function questionTypeFromTool(type: TemplateTool): string {
  if (type === "choice") {
    return "single_choice";
  }
  return type;
}

function normalizeTemplateStatus(status?: string): TemplateStatus {
  if (!status) {
    return "draft";
  }
  if (status === "draft" || status === "disabled") {
    return status;
  }
  return "published";
}

function regionsFromTemplate(template?: PaperTemplate): CanvasRegion[] {
  if (!template) {
    return [];
  }
  return template.questions.map((question) => {
    const type = toolFromQuestionType(question.type);
    return {
      id: question.id,
      no: question.no,
      type,
      label: templateTools[type].label,
      color: templateTools[type].color,
      borderStyle: "solid",
      score: question.score,
      standardAnswer: question.standardAnswer ?? "",
      scoringRules: question.scoringRules ?? [],
      knowledge: question.knowledge,
      region: question.region
    };
  });
}

function hasDashboardData(data: DashboardData) {
  return data.metrics.length > 0
    || data.scanQueue.length > 0
    || data.reviewQueue.length > 0
    || data.weakPoints.length > 0
    || data.homeworkWatch.length > 0;
}

function hasAnalyticsData(data: ClassroomAnalytics) {
  return data.questionStats.length > 0
    || data.knowledgeStats.length > 0
    || data.studentRisks.length > 0
    || data.studentScores.length > 0
    || data.scoreBands.length > 0
    || data.objectiveExceptions.length > 0;
}

function normalizeDashboardData(data: Partial<DashboardData> | null): DashboardData {
  const source = data?.source === "fixtures" || data?.source === "local" ? data.source : "database";
  return {
    source,
    metrics: Array.isArray(data?.metrics) ? data.metrics : [],
    scanQueue: Array.isArray(data?.scanQueue) ? data.scanQueue : [],
    reviewQueue: Array.isArray(data?.reviewQueue) ? data.reviewQueue : [],
    weakPoints: Array.isArray(data?.weakPoints) ? data.weakPoints : [],
    homeworkWatch: Array.isArray(data?.homeworkWatch) ? data.homeworkWatch : []
  };
}

function normalizeAnalyticsData(data: Partial<ClassroomAnalytics> | null): ClassroomAnalytics {
  return {
    className: data?.className ?? "未选择班级",
    averageScore: data?.averageScore ?? 0,
    highestScore: data?.highestScore ?? 0,
    lowestScore: data?.lowestScore ?? 0,
    studentCount: data?.studentCount ?? 0,
    gradedCount: data?.gradedCount ?? 0,
    completionRate: data?.completionRate ?? 0,
    passRate: data?.passRate ?? 0,
    excellentRate: data?.excellentRate ?? 0,
    questionStats: Array.isArray(data?.questionStats) ? data.questionStats : [],
    questionDetails: Array.isArray(data?.questionDetails) ? data.questionDetails : [],
    knowledgeStats: Array.isArray(data?.knowledgeStats) ? data.knowledgeStats : [],
    studentRisks: Array.isArray(data?.studentRisks) ? data.studentRisks : [],
    studentScores: Array.isArray(data?.studentScores) ? data.studentScores : [],
    scoreBands: Array.isArray(data?.scoreBands) ? data.scoreBands : [],
    objectiveExceptions: Array.isArray(data?.objectiveExceptions) ? data.objectiveExceptions : []
  };
}

function nextLoadingState(current: RequestState, loadingMessage: string, processingMessage: string): RequestState {
  if (current.status === "success" || current.status === "error" || current.status === "empty") {
    return { status: "processing", message: processingMessage };
  }
  return { status: "loading", message: loadingMessage };
}

function includesSearch(value: string, search: string) {
  return value.toLowerCase().includes(search.trim().toLowerCase());
}

function pageItems<T>(items: T[], page: number) {
  return items.slice((page - 1) * pageSize, page * pageSize);
}

function uniqueOptions(values: string[]): Option[] {
  return Array.from(new Set(values.filter(Boolean))).map((value) => ({ label: value, value }));
}

function clampScore(value: number, fullScore: number) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.min(fullScore, Math.max(0, Math.round(value * 2) / 2));
}

function validateScore(value: number, fullScore: number) {
  if (!Number.isFinite(value)) {
    return "请输入有效分数";
  }
  if (value < 0 || value > fullScore) {
    return `分数必须在 0 到 ${fullScore} 之间`;
  }
  if (Math.round(value * 2) !== value * 2) {
    return "当前仅支持 0.5 分步进";
  }
  return "";
}

function reviewStageLabel(stage?: string) {
  if (stage === "second_review") {
    return "二审";
  }
  if (stage === "arbitration") {
    return "仲裁";
  }
  if (stage === "spot_check") {
    return "抽检";
  }
  return "一审";
}

function reviewStatusLabel(status?: string) {
  if (status === "second_review") {
    return "二审中";
  }
  if (status === "arbitration") {
    return "仲裁中";
  }
  if (status === "reviewed") {
    return "已裁定";
  }
  return "待复核";
}

function decisionLabel(action: string) {
  const labels: Record<string, string> = {
    accepted_ai: "接受 AI",
    modified: "修改保存",
    rejected: "驳回建议",
    second_review: "提交二审",
    arbitration: "提交仲裁",
    spot_check: "抽检确认"
  };
  return labels[action] ?? action;
}

function formatFileSize(size: number) {
  if (size >= 1024 * 1024) {
    return `${(size / 1024 / 1024).toFixed(1)} MB`;
  }
  return `${Math.max(1, Math.round(size / 1024))} KB`;
}

function scanQueueLabel(status?: string) {
  if (status === "queued") {
    return "已入队";
  }
  if (status === "failed") {
    return "入队失败";
  }
  if (status === "pending") {
    return "等待入队";
  }
  return "未入队";
}

function scanMatchLabel(status?: string) {
  if (status === "matched") {
    return "已匹配";
  }
  if (status === "conflict") {
    return "待人工确认";
  }
  return "待匹配";
}

function scanFileValidationError(files: File[]) {
  if (files.length === 0) {
    return "请选择 PDF、图片或 ZIP 扫描包";
  }
  if (files.length > maxScanFileCount) {
    return `一次最多选择 ${maxScanFileCount} 个文件`;
  }
  for (const file of files) {
    const extension = `.${file.name.split(".").pop()?.toLowerCase() ?? ""}`;
    if (!allowedScanExtensions.includes(extension)) {
      return `${file.name} 格式不支持，请选择 PDF、PNG、JPG、WebP 或 ZIP`;
    }
    if (file.type && !allowedScanMimeTypes.includes(file.type)) {
      return `${file.name} 类型不支持：${file.type}`;
    }
    if (file.size > maxScanFileSizeBytes) {
      return `${file.name} 超过 ${formatFileSize(maxScanFileSizeBytes)} 限制`;
    }
  }
  return "";
}

function RequestStateView({
  state,
  onRetry,
  compact = false
}: {
  state: RequestState;
  onRetry?: () => void;
  compact?: boolean;
}) {
  if (state.status === "success") {
    return null;
  }
  const icon = state.status === "loading" || state.status === "processing"
    ? <Loader2 size={18} />
    : <AlertCircle size={18} />;
  return (
    <div className={`request-state request-state-${state.status}${compact ? " compact" : ""}`} role={state.status === "error" ? "alert" : "status"}>
      <div className="request-state-copy">
        {icon}
        <div>
          <strong>{state.message}</strong>
          {state.detail ? <span>{state.detail}</span> : null}
        </div>
      </div>
      {(state.status === "error" || state.status === "empty") && onRetry ? (
        <button className="ghost-button" onClick={onRetry} type="button">
          <RefreshCw size={16} />重试
        </button>
      ) : null}
    </div>
  );
}

function apiStatusFromRequests(states: RequestState[]): ConnectionStatus {
  if (states.some((state) => state.status === "error")) {
    return "unavailable";
  }
  if (states.some((state) => state.status === "loading" || state.status === "processing")) {
    return "checking";
  }
  return "available";
}

function connectionLabel(status: ConnectionStatus) {
  const labels = {
    checking: "检测中",
    available: "可用",
    unavailable: "不可用",
    skipped: "已跳过"
  } satisfies Record<ConnectionStatus, string>;
  return labels[status];
}

function ConnectionStatusBar({
  apiStatus,
  databaseStatus = "skipped",
  storageStatus = "skipped",
  note
}: {
  apiStatus: ConnectionStatus;
  databaseStatus?: ConnectionStatus;
  storageStatus?: ConnectionStatus;
  note: string;
}) {
  const items = [
    { key: "api", icon: <Sparkles size={16} />, label: "Go API", status: apiStatus },
    { key: "database", icon: <Database size={16} />, label: "数据库", status: databaseStatus },
    { key: "storage", icon: <Cloud size={16} />, label: "对象存储", status: storageStatus }
  ];
  return (
    <div className="connection-bar" role="status">
      <div className="connection-items">
        {items.map((item) => (
          <span className={`connection-pill connection-${item.status}`} key={item.key}>
            {item.icon}
            {item.label}
            <strong>{connectionLabel(item.status)}</strong>
          </span>
        ))}
      </div>
      <span className="connection-note">{note}</span>
    </div>
  );
}

function TableToolbar({
  searchValue,
  onSearchChange,
  searchPlaceholder,
  filterValue,
  onFilterChange,
  filterOptions,
  sortValue,
  onSortChange,
  sortOptions,
  selectedCount,
  totalCount,
  batchLabel,
  onBatchAction
}: {
  searchValue: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder: string;
  filterValue: string;
  onFilterChange: (value: string) => void;
  filterOptions: Option[];
  sortValue: string;
  onSortChange: (value: string) => void;
  sortOptions: Option[];
  selectedCount: number;
  totalCount: number;
  batchLabel: string;
  onBatchAction: () => void;
}) {
  return (
    <div className="table-toolbar">
      <label className="table-search">
        搜索
        <input
          onChange={(event: { target: { value: string } }) => onSearchChange(event.target.value)}
          placeholder={searchPlaceholder}
          value={searchValue}
        />
      </label>
      <label>
        筛选
        <select onChange={(event: { target: { value: string } }) => onFilterChange(event.target.value)} value={filterValue}>
          {filterOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      </label>
      <label>
        排序
        <select onChange={(event: { target: { value: string } }) => onSortChange(event.target.value)} value={sortValue}>
          {sortOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      </label>
      <div className="batch-actions">
        <span>已选 {selectedCount} / {totalCount}</span>
        <button className="ghost-button" disabled={selectedCount === 0} onClick={onBatchAction} type="button">{batchLabel}</button>
      </div>
    </div>
  );
}

function TablePagination({
  page,
  total,
  onPageChange
}: {
  page: number;
  total: number;
  onPageChange: (page: number) => void;
}) {
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="table-pagination">
      <span>第 {Math.min(page, pageCount)} / {pageCount} 页</span>
      <div>
        <button className="ghost-button" disabled={page <= 1} onClick={() => onPageChange(page - 1)} type="button">上一页</button>
        <button className="ghost-button" disabled={page >= pageCount} onClick={() => onPageChange(page + 1)} type="button">下一页</button>
      </div>
    </div>
  );
}

function loadStoredDrafts(): TemplateDraft[] {
  try {
    const raw = window.localStorage.getItem(templateDraftStorageKey);
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function loadStoredTemplateLibrary(): PaperTemplate[] {
  try {
    const raw = window.localStorage.getItem(templateLibraryStorageKey);
    if (!raw) {
      return fallbackTemplates;
    }
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) && parsed.length > 0 ? parsed : fallbackTemplates;
  } catch {
    return fallbackTemplates;
  }
}

function PortalView({ role, onNotice }: { role: "student" | "guardian"; onNotice: (value: string) => void }) {
  const [data, setData] = useState<PortalData | null>(null);
  const [children, setChildren] = useState<Array<{ studentId: string; studentName: string }>>([]);
  const [loading, setLoading] = useState(true);

  async function loadPortal(studentId = "") {
    setLoading(true);
    try {
      const path = role === "guardian"
        ? `/api/portal/guardian?guardianId=guardian_001${studentId ? `&studentId=${studentId}` : ""}`
        : "/api/portal/student?studentId=stu_001";
      const response = await fetch(path);
      if (!response.ok) throw new Error("portal unavailable");
      const payload = await response.json() as PortalData | { children: Array<{ studentId: string; studentName: string }>; selected: PortalData };
      if ("selected" in payload) {
        setChildren(payload.children);
        setData(payload.selected);
      } else {
        setData(payload);
      }
    } catch {
      setData({
        studentId: "stu_001", studentName: "张三", gradeName: "六年级", className: "六年级 3 班",
        scoreSummary: { highest: 96, lowest: 62, average: 81.6, personal: 85 },
        homework: [{ id: "assign_001", title: "六年级数学期中卷", subject: "数学", status: "graded", dueAt: "2026-06-23 18:00" }, { id: "assign_002", title: "分数应用题专项", subject: "数学", status: "pending", dueAt: "2026-06-25 18:00" }],
        scoreTrend: [{ label: "单元测验", score: 76 }, { label: "月考", score: 81 }, { label: "期中", score: 85 }],
        mistakes: [{ subject: "数学", paperCount: 3, homeworkCount: 0, items: [] }],
        ai: [{ key: "analysis", name: "AI 学情分析", status: "planned", description: "多维分析学科与知识点短板，输出补漏地图" }, { key: "ladder", name: "天梯攻略", status: "planned", description: "根据补漏地图生成阶段练习册并周期复核" }]
      });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void loadPortal(); }, [role]);

  async function reserve(capability: string) {
    try {
      await fetch(`/api/ai/capabilities/${capability}/requests`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ studentId: data?.studentId, channel: role }) });
      onNotice("已登记体验意向，能力接入后会通知你");
    } catch {
      onNotice("体验意向已在本地记录");
    }
  }

  if (loading || !data) return <RequestStateView state={{ status: "loading", message: "正在加载个人学情" }} />;
  return (
    <section className="portal-stack">
      <div className="portal-hero">
        <div><p className="eyebrow">{role === "guardian" ? "家长学情视野" : "我的学习工作台"}</p><h2>{data.studentName}，{role === "guardian" ? "本周学习节奏稳定" : "继续保持上升趋势"}</h2><span>{data.gradeName} · {data.className}</span></div>
        {role === "guardian" && children.length > 1 ? <label>查看孩子<select value={data.studentId} onChange={(event: { target: { value: string } }) => void loadPortal(event.target.value)}>{children.map((child) => <option key={child.studentId} value={child.studentId}>{child.studentName}</option>)}</select></label> : null}
      </div>
      <div className="portal-score-grid">
        {[{ label: "我的成绩", value: data.scoreSummary.personal }, { label: "年级/班级最高分", value: data.scoreSummary.highest }, { label: "最低分", value: data.scoreSummary.lowest }, { label: "平均分", value: data.scoreSummary.average }].map((item) => <article key={item.label}><span>{item.label}</span><strong>{item.value}</strong><small>{data.className}</small></article>)}
      </div>
      <div className="portal-two-column">
        <article className="panel portal-panel"><div className="panel-head"><div><p className="eyebrow">Homework</p><h3>作业完成情况</h3></div><span>{data.homework.filter((item) => item.status !== "pending").length}/{data.homework.length} 已完成</span></div><div className="portal-list">{data.homework.map((item) => <div key={item.id}><span className={`status-dot ${item.status}`} /><div><strong>{item.title}</strong><small>{item.subject} · 截止 {item.dueAt || "未设置"}</small></div><em>{item.status === "pending" ? "待完成" : "已完成"}</em></div>)}</div></article>
        <article className="panel portal-panel"><div className="panel-head"><div><p className="eyebrow">Score Trend</p><h3>成绩趋势</h3></div><strong className="trend-up">+{Math.max(0, data.scoreTrend[data.scoreTrend.length - 1].score - data.scoreTrend[0].score)} 分</strong></div><div className="trend-chart">{data.scoreTrend.map((point) => <div key={point.label}><span style={{ height: `${point.score}%` }} /><strong>{point.score}</strong><small>{point.label}</small></div>)}</div></article>
      </div>
      <article className="panel portal-panel"><div className="panel-head"><div><p className="eyebrow">Mistake Book</p><h3>学科错题集</h3></div><button className="ghost-button" type="button">查看全部错题</button></div><div className="subject-mistake-grid">{data.mistakes.map((subject) => <div key={subject.subject}><BookOpenCheck size={22} /><strong>{subject.subject}</strong><span>往期试卷 {subject.paperCount} 题</span><span>作业 {subject.homeworkCount} 题</span></div>)}</div></article>
      <div className="ai-product-grid">{data.ai.map((item, index) => <article className={index === 0 ? "ai-product primary" : "ai-product"} key={item.key}><div><Sparkles size={22} /><span>即将开放</span></div><h3>{item.name}</h3><p>{item.description}</p>{role === "guardian" ? <small>先看清短板，再决定是否购买个性化提升服务。</small> : <small>模型厂商接入后开放，不生成虚假分析。</small>}<button onClick={() => void reserve(item.key)} type="button">登记体验意向</button></article>)}</div>
    </section>
  );
}

function OrganizationView({ onNotice }: { onNotice: (value: string) => void }) {
  const [graph, setGraph] = useState<OrganizationGraph | null>(null);
  const [kind, setKind] = useState("schools");
  const [name, setName] = useState("");
  async function load() { try { const response = await fetch("/api/organization/graph"); if (response.ok) setGraph(await response.json() as OrganizationGraph); } catch { setGraph(null); } }
  useEffect(() => { void load(); }, []);
  async function create() {
    if (!name.trim()) { onNotice("请填写名称"); return; }
    const schoolId = graph?.schools[0]?.id ?? "school_001";
    const gradeId = graph?.schools[0]?.children?.[0]?.id ?? "grade_6";
    const classId = graph?.schools[0]?.children?.[0]?.children?.[0]?.id ?? "class_603";
    const payload = { name, schoolId, gradeId, classId, subjectId: "subject_math", stage: "primary" };
    const response = await fetch(`/api/organization/${kind}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
    if (!response.ok) { onNotice("创建失败，请检查数据库和关联信息"); return; }
    setName(""); onNotice("组织数据已创建"); void load();
  }
  return <section className="organization-layout"><article className="panel"><div className="panel-head"><div><p className="eyebrow">Organization Graph</p><h2>学校组织关系</h2></div><button className="ghost-button" onClick={() => void load()} type="button"><RefreshCw size={16} />刷新</button></div><div className="organization-counts">{Object.entries(graph?.counts ?? {}).map(([key, value]) => <div key={key}><strong>{value}</strong><span>{{ schools: "学校", grades: "年级", classes: "班级", teachers: "教师", students: "学生", subjects: "学科", pendingCertifications: "待认证" }[key] ?? key}</span></div>)}</div><div className="org-tree">{graph?.schools.map((school) => <div key={school.id}><strong>{school.name}</strong>{school.children?.map((grade) => <div key={grade.id}><span>{grade.name}</span>{grade.children?.map((item) => <small key={item.id}>{item.name}</small>)}</div>)}</div>) ?? <p>连接数据库后展示组织树。</p>}</div></article><aside className="panel org-create"><p className="eyebrow">Create & Link</p><h3>新增组织成员</h3><label>类型<select value={kind} onChange={(event: { target: { value: string } }) => setKind(event.target.value)}><option value="schools">学校</option><option value="grades">年级</option><option value="subjects">学科</option><option value="classes">班级</option><option value="teachers">教师</option><option value="students">学生</option></select></label><label>名称<input value={name} onChange={(event: { target: { value: string } }) => setName(event.target.value)} placeholder="请输入名称" /></label><button className="primary-button" onClick={() => void create()} type="button">创建并关联</button><div className="org-flow"><strong>家长访问门槛</strong><span>教师发邀请链接</span><span>家长提交认证</span><span>管理员审批通过</span><small>审批前不会写入学生—家长访问关系。</small></div></aside></section>;
}

function App() {
  const [dashboard, setDashboard] = useState<DashboardData>(fallbackDashboard);
  const [subjective, setSubjective] = useState<SubjectiveData | null>(fallbackSubjective);
  const [templates, setTemplates] = useState<PaperTemplate[]>(loadStoredTemplateLibrary);
  const [analytics, setAnalytics] = useState<ClassroomAnalytics>(fallbackAnalytics);
  const [dashboardState, setDashboardState] = useState<RequestState>({ status: "loading", message: "正在加载工作台数据" });
  const [subjectiveState, setSubjectiveState] = useState<RequestState>({ status: "loading", message: "正在连接阅卷队列" });
  const [templatesState, setTemplatesState] = useState<RequestState>({ status: "loading", message: "正在加载模版库" });
  const [analyticsState, setAnalyticsState] = useState<RequestState>({ status: "loading", message: "正在加载学情数据" });
  const [selectedTemplateId, setSelectedTemplateId] = useState(fallbackTemplates[0].id);
  const [selectedReviewId, setSelectedReviewId] = useState(fallbackSubjective.reviewId);
  const [activeView, setActiveView] = useState<ActiveView>("workspace");
  const [currentRole, setCurrentRole] = useState<UserRole>("teacher");
  const [overlay, setOverlay] = useState<Overlay>(null);
  const [notice, setNotice] = useState("已连接本地开发环境");
  const [scanTitle, setScanTitle] = useState("六年级数学期中卷");
  const [scanClassName, setScanClassName] = useState("六年级 3 班");
  const [scanPages, setScanPages] = useState(48);
  const [scanTemplateId, setScanTemplateId] = useState(fallbackTemplates[0].id);
  const [scanNotes, setScanNotes] = useState("");
  const [scanFiles, setScanFiles] = useState<File[]>([]);
  const [scanUploadedFiles, setScanUploadedFiles] = useState<ScanUploadFile[]>([]);
  const [scanFileError, setScanFileError] = useState("");
  const [templateSourceMode, setTemplateSourceMode] = useState<TemplateSourceMode>("library");
  const [paperSources, setPaperSources] = useState<TemplatePaperSource[]>(fallbackPaperSources);
  const [selectedPaperSourceId, setSelectedPaperSourceId] = useState("");
  const [canvasTitle, setCanvasTitle] = useState("未命名试卷模板");
  const [canvasSize, setCanvasSize] = useState<CanvasSize>(canvasPresets[1]);
  const [canvasZoom, setCanvasZoom] = useState(1);
  const [activeTool, setActiveTool] = useState<TemplateTool>("subjective");
  const [canvasRegions, setCanvasRegions] = useState<CanvasRegion[]>([]);
  const [selectedRegionId, setSelectedRegionId] = useState("");
  const [templateDrafts, setTemplateDrafts] = useState<TemplateDraft[]>(loadStoredDrafts);
  const [dragState, setDragState] = useState<DragState | null>(null);
  const [canvasElement, setCanvasElement] = useState<HTMLDivElement | null>(null);
  const [score, setScore] = useState(fallbackSubjective.ai.score);
  const [note, setNote] = useState("步骤完整，结果正确，表达略不规范。");
  const [savedState, setSavedState] = useState("未保存");
  const [queueStatus, setQueueStatus] = useState("正在连接阅卷队列");
  const [isReviewLoading, setIsReviewLoading] = useState(false);
  const [activeMode, setActiveMode] = useState<"review" | "template">("review");
  const [scanSearch, setScanSearch] = useState("");
  const [scanFilter, setScanFilter] = useState("all");
  const [scanSort, setScanSort] = useState("progress_desc");
  const [scanPage, setScanPage] = useState(1);
  const [selectedScanIds, setSelectedScanIds] = useState<string[]>([]);
  const [scanPreviewTask, setScanPreviewTask] = useState<ScanJob | null>(null);
  const [scanPreviewFiles, setScanPreviewFiles] = useState<ScanUploadFile[]>([]);
  const [scanMatchNames, setScanMatchNames] = useState<Record<string, string>>({});
  const [aiSuggestedRegions, setAiSuggestedRegions] = useState<CanvasRegion[]>([]);
  const [aiSuggestionState, setAiSuggestionState] = useState<RequestState>({ status: "empty", message: "暂无 AI 拆卷建议" });
  const [reviewSearch, setReviewSearch] = useState("");
  const [reviewFilter, setReviewFilter] = useState("all");
  const [reviewSort, setReviewSort] = useState("confidence_desc");
  const [reviewPage, setReviewPage] = useState(1);
  const [selectedReviewIds, setSelectedReviewIds] = useState<string[]>([]);
  const [reviewClassFilter, setReviewClassFilter] = useState("all");
  const [reviewPaperFilter, setReviewPaperFilter] = useState("all");
  const [reviewQuestionFilter, setReviewQuestionFilter] = useState("all");
  const [reviewStatusFilter, setReviewStatusFilter] = useState("all");
  const [reviewStage, setReviewStage] = useState("first_review");
  const [gradingHistory, setGradingHistory] = useState<GradingHistoryItem[]>([]);
  const [paperZoom, setPaperZoom] = useState(1);
  const [paperRotation, setPaperRotation] = useState(0);
  const [paperOffset, setPaperOffset] = useState({ x: 0, y: 0 });
  const [paperDragStart, setPaperDragStart] = useState<{ x: number; y: number; ox: number; oy: number } | null>(null);
  const [templateSearch, setTemplateSearch] = useState("");
  const [templateFilter, setTemplateFilter] = useState("all");
  const [templateSort, setTemplateSort] = useState("name_asc");
  const [templatePage, setTemplatePage] = useState(1);
  const [selectedTemplateIds, setSelectedTemplateIds] = useState<string[]>([]);
  const [mistakeSearch, setMistakeSearch] = useState("");
  const [mistakeFilter, setMistakeFilter] = useState("all");
  const [mistakeSort, setMistakeSort] = useState("wrong_desc");
  const [mistakePage, setMistakePage] = useState(1);
  const [selectedMistakeIds, setSelectedMistakeIds] = useState<string[]>([]);
  const [wrongQuestions, setWrongQuestions] = useState<WrongQuestion[]>(fallbackWrongQuestions);
  const [mistakesState, setMistakesState] = useState<RequestState>({ status: "loading", message: "正在加载错题档案" });
  const [selectedMistake, setSelectedMistake] = useState<WrongQuestion | null>(fallbackWrongQuestions[0]);
  const [mistakePaperFilter, setMistakePaperFilter] = useState("all");
  const [mistakeClassFilter, setMistakeClassFilter] = useState("all");
  const [mistakeStudentFilter, setMistakeStudentFilter] = useState("all");
  const [mistakeKnowledgeFilter, setMistakeKnowledgeFilter] = useState("all");
  const [learningProfile, setLearningProfile] = useState<LearningProfile>(fallbackLearningProfile);
  const [guardianReport, setGuardianReport] = useState<GuardianReport>(fallbackGuardianReport);

  const publishedTemplates = useMemo(
    () => templates.filter((template) => normalizeTemplateStatus(template.status) === "published"),
    [templates]
  );

  useEffect(() => {
    loadDashboard();
    loadReviewQueue(true);
    loadSubjective();
    loadTemplates();
    loadAnalytics();
    loadWrongQuestions();
    loadLearningProfile();
  }, []);

  useEffect(() => {
    if (!publishedTemplates.some((template) => template.id === scanTemplateId)) {
      setScanTemplateId(publishedTemplates[0]?.id ?? "");
    }
  }, [publishedTemplates, scanTemplateId]);

  const selectedTemplate = useMemo(
    () => selectedTemplateId ? templates.find((item) => item.id === selectedTemplateId) ?? templates[0] : undefined,
    [templates, selectedTemplateId]
  );

  const selectedTemplateQuestion = useMemo(() => {
    if (!selectedTemplate) {
      return undefined;
    }
    return selectedTemplate.questions.find((item) => item.id === subjective?.questionId)
      ?? selectedTemplate.questions.find((item) => item.type === "subjective")
      ?? selectedTemplate.questions[0];
  }, [selectedTemplate, subjective?.questionId]);

  const selectedReview = useMemo(
    () => dashboard.reviewQueue.find((item) => item.id === selectedReviewId) ?? dashboard.reviewQueue[0],
    [dashboard.reviewQueue, selectedReviewId]
  );

  const selectedCanvasRegion = useMemo(
    () => canvasRegions.find((item) => item.id === selectedRegionId),
    [canvasRegions, selectedRegionId]
  );

  const selectedTemplateStatus = normalizeTemplateStatus(selectedTemplate?.status);
  const canEditSelectedTemplate = selectedTemplateStatus === "draft";

  const selectedPaperSource = useMemo(
    () => paperSources.find((item) => item.id === selectedPaperSourceId),
    [paperSources, selectedPaperSourceId]
  );

  const activeRole = roleConfig[currentRole];

  const visibleNavItems = useMemo(
    () => navItems.filter((item) => activeRole.views.includes(item.view)),
    [activeRole]
  );

  const can = (permission: Permission) => activeRole.permissions.includes(permission);

  useEffect(() => {
    if (!activeRole.permissions.includes("scan:create") || (activeView !== "workspace" && activeView !== "scan")) {
      return;
    }
    void loadScanTasks(true);
    const timer = window.setInterval(() => {
      void loadScanTasks(true);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [activeRole.permissions, activeView]);

  const filteredScanQueue = useMemo(() => {
    const rows = dashboard.scanQueue
      .filter((job) => scanFilter === "all" || job.status.includes(scanFilter))
      .filter((job) => includesSearch(`${job.title} ${job.className} ${job.status}`, scanSearch));
    return [...rows].sort((a, b) => {
      if (scanSort === "progress_asc") {
        return a.progress - b.progress;
      }
      if (scanSort === "pages_desc") {
        return b.pages - a.pages;
      }
      return b.progress - a.progress;
    });
  }, [dashboard.scanQueue, scanSearch, scanFilter, scanSort]);

  const pagedScanQueue = pageItems(filteredScanQueue, scanPage);

  const filteredReviewQueue = useMemo(() => {
    const rows = dashboard.reviewQueue
      .filter((item) => {
        if (reviewFilter === "all") {
          return true;
        }
        return reviewFilter === "high" ? item.confidence >= 80 : item.confidence < 80;
      })
      .filter((item) => reviewClassFilter === "all" || (item.className ?? "") === reviewClassFilter)
      .filter((item) => reviewPaperFilter === "all" || item.paperName === reviewPaperFilter)
      .filter((item) => reviewQuestionFilter === "all" || item.questionNo === reviewQuestionFilter)
      .filter((item) => reviewStatusFilter === "all" || (item.status ?? "pending") === reviewStatusFilter)
      .filter((item) => includesSearch(`${item.studentName} ${item.paperName} ${item.questionNo}`, reviewSearch));
    return [...rows].sort((a, b) => {
      if (reviewSort === "question_asc") {
        return Number(a.questionNo) - Number(b.questionNo);
      }
      if (reviewSort === "student_asc") {
        return a.studentName.localeCompare(b.studentName, "zh-Hans-CN");
      }
      return b.confidence - a.confidence;
    });
  }, [dashboard.reviewQueue, reviewSearch, reviewFilter, reviewSort, reviewClassFilter, reviewPaperFilter, reviewQuestionFilter, reviewStatusFilter]);

  const pagedReviewQueue = pageItems(filteredReviewQueue, reviewPage);
  const reviewClassOptions = useMemo(() => uniqueOptions(dashboard.reviewQueue.map((item) => item.className ?? "").filter(Boolean)), [dashboard.reviewQueue]);
  const reviewPaperOptions = useMemo(() => uniqueOptions(dashboard.reviewQueue.map((item) => item.paperName)), [dashboard.reviewQueue]);
  const reviewQuestionOptions = useMemo(() => uniqueOptions(dashboard.reviewQueue.map((item) => item.questionNo)).sort((a, b) => Number(a.value) - Number(b.value)), [dashboard.reviewQueue]);
  const selectedReviewIndex = filteredReviewQueue.findIndex((item) => item.id === selectedReviewId);
  const scoreError = subjective
    ? validateScore(score, subjective.fullScore)
    : "";

  useEffect(() => {
    if (activeView !== "grading" || !subjective || !can("grading:decide")) {
      return;
    }
    const currentSubjective = subjective;
    function handleShortcut(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target && ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)) {
        return;
      }
      const key = event.key.toLowerCase();
      if (key === "a") {
        event.preventDefault();
        acceptAIScore();
      }
      if (key === "m") {
        event.preventDefault();
        void saveDecision("modified");
      }
      if (key === "r") {
        event.preventDefault();
        void saveDecision("rejected");
      }
      if (key === "f") {
        event.preventDefault();
        updateScore(currentSubjective.fullScore);
      }
      if (key === "z") {
        event.preventDefault();
        updateScore(0);
      }
      if (key === "n") {
        event.preventDefault();
        void openAdjacentReview(1);
      }
      if (key === "b") {
        event.preventDefault();
        void openAdjacentReview(-1);
      }
    }
    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [activeView, subjective, score, note, reviewStage, filteredReviewQueue, selectedReviewIndex, scoreError, currentRole]);

  const filteredTemplates = useMemo(() => {
    const rows = templates
      .filter((template) => {
        if (templateFilter === "all") {
          return true;
        }
        if (templateFilter.startsWith("status:")) {
          return normalizeTemplateStatus(template.status) === templateFilter.replace("status:", "");
        }
        return template.subject === templateFilter || template.grade === templateFilter;
      })
      .filter((template) => includesSearch(`${template.name} ${template.grade} ${template.subject}`, templateSearch));
    return [...rows].sort((a, b) => {
      if (templateSort === "score_desc") {
        return b.totalScore - a.totalScore;
      }
      if (templateSort === "questions_desc") {
        return b.questionCount - a.questionCount;
      }
      return a.name.localeCompare(b.name, "zh-Hans-CN");
    });
  }, [templates, templateSearch, templateFilter, templateSort]);

  const pagedTemplates = pageItems(filteredTemplates, templatePage);

  const filteredMistakes = useMemo(() => {
    const rows = wrongQuestions
      .filter((item) => mistakeFilter === "all" || item.errorType === mistakeFilter)
      .filter((item) => mistakePaperFilter === "all" || item.sourcePaper === mistakePaperFilter)
      .filter((item) => mistakeClassFilter === "all" || item.className === mistakeClassFilter)
      .filter((item) => mistakeStudentFilter === "all" || item.studentName === mistakeStudentFilter)
      .filter((item) => mistakeKnowledgeFilter === "all" || item.knowledgePoint === mistakeKnowledgeFilter)
      .filter((item) => includesSearch(`${item.questionNo} ${item.questionType} ${item.studentName} ${item.wrongReason}`, mistakeSearch));
    return [...rows].sort((a, b) => {
      if (mistakeSort === "question_asc") {
        return Number(a.questionNo) - Number(b.questionNo);
      }
      if (mistakeSort === "student_asc") {
        return a.studentName.localeCompare(b.studentName, "zh-Hans-CN");
      }
      return (a.score / Math.max(a.maxScore, 1)) - (b.score / Math.max(b.maxScore, 1));
    });
  }, [wrongQuestions, mistakeSearch, mistakeFilter, mistakeSort, mistakePaperFilter, mistakeClassFilter, mistakeStudentFilter, mistakeKnowledgeFilter]);

  const pagedMistakes = pageItems(filteredMistakes, mistakePage);

  const activeConnection = useMemo(() => {
    const requestStatesByView = {
      workspace: [dashboardState, subjectiveState, analyticsState],
      organization: [dashboardState],
      scan: [dashboardState],
      templates: [templatesState],
      grading: [subjectiveState],
      mistakes: [mistakesState, analyticsState],
      analytics: [analyticsState]
    } satisfies Record<ActiveView, RequestState[]>;
    const apiStatus = apiStatusFromRequests(requestStatesByView[activeView]);
    const notes = {
      workspace: "数据库和对象存储连接检测暂时跳过；工作台优先展示 API 数据，失败时回退演示数据。",
      organization: "组织关系、教师关联和家长认证均通过 Go API 持久化。",
      scan: "扫描上传、任务创建和进度轮询优先走 Go API；API 不可用时保留当前页面数据。",
      templates: "数据库连接检测暂时跳过；模板库优先读取 API，失败时回退本地模板。",
      grading: "数据库连接检测暂时跳过；教师裁定保存失败时只保留本地状态提示。",
      mistakes: "数据库连接检测暂时跳过；错题统计优先读取学情 API，失败时回退演示数据。",
      analytics: "数据库连接检测暂时跳过；学情分析优先读取 API，失败时回退演示数据。"
    } satisfies Record<ActiveView, string>;
    const dashboardSourceNote = dashboard.source === "database"
      ? "工作台正在展示 Go API 从数据库读取的实时数据。"
      : dashboard.source === "fixtures"
        ? "Go API 已响应，但数据库查询不可用；工作台展示后端演示数据。"
        : "工作台 API 当前不可用，页面已使用本地演示数据兜底。";
    const apiNote = apiStatus === "unavailable"
      ? "API 当前不可用，页面已使用本地演示数据兜底。"
      : apiStatus === "checking"
        ? "正在检测当前页面 API 请求状态。"
        : activeView === "workspace" || activeView === "scan"
          ? dashboardSourceNote
          : notes[activeView];
    const databaseStatus = activeView === "workspace" || activeView === "scan"
      ? dashboard.source === "database"
        ? "available"
        : dashboard.source === "fixtures"
          ? "unavailable"
          : "skipped"
      : "skipped";
    return {
      apiStatus,
      databaseStatus: databaseStatus as ConnectionStatus,
      storageStatus: "skipped" as ConnectionStatus,
      note: apiNote
    };
  }, [activeView, dashboard.source, dashboardState, subjectiveState, templatesState, analyticsState, mistakesState]);

  const viewCopy = {
    workspace: { eyebrow: "六年级 3 班 · 今日工作台", title: "先处理阅卷，再看学情" },
    organization: { eyebrow: "Organization & Identity", title: "管理学校、教师、学生与家长认证" },
    scan: { eyebrow: "Scan Import", title: "导入扫描件并进入 OCR 队列" },
    templates: { eyebrow: "Paper Templates", title: "试卷模版" },
    grading: { eyebrow: "Grading Center", title: "主观题左右分屏批阅" },
    mistakes: { eyebrow: "Wrong Questions", title: "沉淀错题和薄弱知识点" },
    analytics: { eyebrow: "Class Analytics", title: "查看班级学情画像" }
  } satisfies Record<ActiveView, { eyebrow: string; title: string }>;

  function openView(view: ActiveView) {
    if (!activeRole.views.includes(view)) {
      const fallbackView = activeRole.views[0] ?? "workspace";
      setActiveView(fallbackView);
      setNotice(`${activeRole.label}角色无权访问该入口`);
      return;
    }
    setActiveView(view);
    setOverlay(null);
    if (view === "grading") {
      setActiveMode("review");
    }
    if (view === "templates") {
      setActiveMode("template");
    }
  }

  function switchRole(role: UserRole) {
    const nextRole = roleConfig[role];
    setCurrentRole(role);
    setOverlay(null);
    if (!nextRole.views.includes(activeView)) {
      setActiveView(nextRole.views[0] ?? "workspace");
    }
    if (!nextRole.permissions.includes("scan:create")) {
      setTemplateSourceMode("library");
    }
    setNotice(`已切换为${nextRole.label}视图`);
  }

  function toggleOverlay(next: Exclude<Overlay, null>) {
    setOverlay((current) => current === next ? null : next);
  }

  function openScanImport() {
    setScanFilter("all");
    setScanSearch("");
    setScanPage(1);
    openView("scan");
    setNotice("已进入扫描导入，可创建新的 OCR 队列任务");
  }

  function openTemplateCreate() {
    if (!can("template:edit")) {
      openView("templates");
      return;
    }
    setSelectedTemplateId("");
    setCanvasTitle("未命名试卷模板");
    setCanvasRegions([]);
    setSelectedRegionId("");
    setTemplateSourceMode("library");
    setCanvasSize(canvasPresets[1]);
    setCanvasZoom(1);
    openView("templates");
    setNotice("已进入模板创建，可选择试卷来源并框选题区");
  }

  function openReviewQueue() {
    setReviewFilter("all");
    setReviewSearch("");
    setReviewPage(1);
    openView("grading");
    void loadReviewQueue(true);
    setNotice("已进入阅卷中心，可从复核队列连续批阅");
  }

  function openMistakeView(filter: "all" | "high" | "low" = "all") {
    setMistakeFilter(filter);
    setMistakeSearch("");
    setMistakePage(1);
    openView("mistakes");
    setNotice(filter === "all" ? "已进入错题集" : "已按错误率筛选错题");
  }

  function openAnalyticsView() {
    openView("analytics");
    setNotice("已进入学情分析，查看班级成绩和学生风险");
  }

  function applyWorkspaceFilter(kind: "class" | "today" | "review") {
    setOverlay(null);
    if (kind === "class") {
      setScanSearch("六年级 3 班");
      setReviewSearch("六年级数学");
      setNotice("已筛选六年级 3 班相关任务");
      return;
    }
    if (kind === "review") {
      setReviewFilter("low");
      setReviewPage(1);
      openReviewQueue();
      setNotice("已进入需重点看的主观题复核队列");
      return;
    }
    setScanFilter("all");
    setReviewFilter("all");
    setNotice("已恢复今日工作台任务视图");
  }

  function openMetricTarget(label: string) {
    if (label.includes("待批") || label.includes("扫描")) {
      openScanImport();
      return;
    }
    if (label.includes("复核")) {
      openReviewQueue();
      return;
    }
    if (label.includes("未提交")) {
      openAnalyticsView();
      setNotice("已定位到学生风险与家长提醒");
      return;
    }
    openAnalyticsView();
  }

  function selectScanFiles(files: FileList | null) {
    const nextFiles = Array.from(files ?? []);
    const error = scanFileValidationError(nextFiles);
    setScanFiles(error ? [] : nextFiles);
    setScanUploadedFiles([]);
    setScanFileError(error);
    if (!error && nextFiles.length > 0) {
      setNotice(`已选择 ${nextFiles.length} 个扫描文件`);
    }
  }

  function validateScanTaskForm() {
    if (!scanTitle.trim()) {
      return "请填写考试/作业名称";
    }
    if (!scanClassName.trim()) {
      return "请填写班级";
    }
    if (!scanTemplateId) {
      return "请选择已发布或可用模板";
    }
    if (!Number.isFinite(scanPages) || scanPages < 1) {
      return "页数必须大于 0";
    }
    const fileError = scanFileValidationError(scanFiles);
    if (fileError) {
      return fileError;
    }
    return "";
  }

  async function uploadScanFilesToApi() {
    const form = new FormData();
    scanFiles.forEach((file) => form.append("files", file));
    const response = await fetch("/api/scan/uploads", {
      method: "POST",
      body: form
    });
    if (!response.ok) {
      throw new Error("scan upload api failed");
    }
    const result = await response.json() as ScanUploadResponse;
    return result.files;
  }

  async function createScanTaskInApi(files: ScanUploadFile[]) {
    const template = templates.find((item) => item.id === scanTemplateId);
    const response = await fetch("/api/scan/tasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        title: scanTitle.trim(),
        className: scanClassName.trim(),
        templateId: scanTemplateId,
        templateVersion: template?.version ?? 1,
        pages: Number(scanPages),
        notes: scanNotes.trim(),
        files
      })
    });
    if (!response.ok) {
      throw new Error("scan task api failed");
    }
    return await response.json() as ScanTaskResponse;
  }

  async function submitScanImport() {
    const formError = validateScanTaskForm();
    if (formError) {
      setScanFileError(formError);
      setNotice(formError);
      return;
    }
    setDashboardState({ status: "processing", message: "正在上传扫描件", detail: "上传成功后会创建 OCR 队列任务。" });
    try {
      const files = await uploadScanFilesToApi();
      setScanUploadedFiles(files);
      const result = await createScanTaskInApi(files);
      setDashboard((current) => ({
        ...current,
        scanQueue: [result.task, ...current.scanQueue]
      }));
      addPaperSourceFromScanJob(result.task);
      setDashboardState({ status: "success", message: "扫描任务已创建" });
      setScanFileError("");
      setScanFiles([]);
      setNotice(result.queueError
        ? `${result.task.title} 已创建任务，但队列投递失败：${result.queueError}`
        : `${result.task.title} 已创建任务并写入队列：${result.queueId ?? result.task.id}`);
      setActiveView("workspace");
      return;
    } catch {
      const localFiles: ScanUploadFile[] = scanFiles.map((file, index) => ({
        key: `local/scan/${Date.now()}_${index}_${file.name}`,
        fileName: file.name,
        contentType: file.type || "application/octet-stream",
        size: file.size,
        url: ""
      }));
      setScanUploadedFiles(localFiles);
      setDashboardState({
        status: "error",
        message: "扫描上传 API 请求失败",
        detail: "已在本地生成临时扫描任务，可稍后重试 API。"
      });
      setScanFileError("");
      setNotice("扫描上传 API 不可用，已生成本地临时任务");
      const nextJob: ScanJob = {
        id: `scan_local_${Date.now()}`,
        title: scanTitle.trim() || "未命名扫描任务",
        className: scanClassName.trim() || "未选择班级",
        templateId: scanTemplateId,
        templateVersion: templates.find((item) => item.id === scanTemplateId)?.version ?? 1,
        pages: Number(scanPages) || 1,
        notes: scanNotes.trim(),
        status: "待 OCR",
        progress: 0,
        failureReason: "",
        retryCount: 0,
        queueStatus: "pending",
        queueMessage: "",
        files: localFiles
      };
      setDashboard((current) => ({
        ...current,
        scanQueue: [nextJob, ...current.scanQueue]
      }));
      addPaperSourceFromScanJob(nextJob);
      setActiveView("workspace");
    }
  }

  function addPaperSourceFromScanJob(job: ScanJob) {
    const nextSource: TemplatePaperSource = {
      id: `paper_local_${Date.now()}`,
      title: job.title,
      className: job.className,
      pages: job.pages,
      size: canvasPresets[1],
      importedAt: "刚刚",
      source: "现场扫描"
    };
    setPaperSources((current) => [nextSource, ...current]);
    return nextSource;
  }

  function applyPaperSource(source: TemplatePaperSource) {
    setSelectedPaperSourceId(source.id);
    setCanvasTitle(source.title);
    setCanvasSize(source.size);
    setNotice(`已载入${source.source}试卷：${source.title}`);
  }

  function importTemplateScanFile(fileName: string) {
    const cleanName = fileName || `${scanTitle} 空白卷`;
    const nextSource: TemplatePaperSource = {
      id: `paper_scan_${Date.now()}`,
      title: cleanName.replace(/\.[^.]+$/, ""),
      className: scanClassName,
      pages: Math.max(1, Number(scanPages) || 1),
      size: canvasSize,
      importedAt: "刚刚",
      source: "现场扫描"
    };
    setPaperSources((current) => [nextSource, ...current]);
    setTemplateSourceMode("scan");
    applyPaperSource(nextSource);
  }

  function importTemplateScanFromCurrentTask() {
    importTemplateScanFile(`${scanTitle} 导入扫描件.png`);
  }

  function persistDrafts(nextDrafts: TemplateDraft[]) {
    setTemplateDrafts(nextDrafts);
    window.localStorage.setItem(templateDraftStorageKey, JSON.stringify(nextDrafts));
  }

  function persistTemplateLibrary(nextTemplates: PaperTemplate[]) {
    setTemplates(nextTemplates);
    window.localStorage.setItem(templateLibraryStorageKey, JSON.stringify(nextTemplates));
  }

  function canvasRegionsToQuestions(regions: CanvasRegion[], _templateID: string): QuestionTemplate[] {
    return regions.map((item, index) => ({
      id: item.id,
      no: item.no || `${index + 1}`,
      type: questionTypeFromTool(item.type),
      score: item.score,
      standardAnswer: item.standardAnswer,
      scoringRules: item.scoringRules,
      knowledge: item.knowledge,
      region: item.region
    }));
  }

  async function writeTemplateToApi(endpoint: string, method: "POST" | "PUT", template: PaperTemplate) {
    const response = await fetch(endpoint, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(template)
    });
    if (!response.ok) {
      throw new Error("template mutation api failed");
    }
    const result = await response.json() as TemplateMutationResponse;
    return result.template;
  }

  async function writeTemplateRegionToApi(endpoint: string, method: "POST" | "PUT", question: QuestionTemplate) {
    const response = await fetch(endpoint, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(question)
    });
    if (!response.ok) {
      throw new Error("template region api failed");
    }
    return await response.json() as TemplateRegionMutationResponse;
  }

  function applyTemplateMutation(template: PaperTemplate) {
    const nextTemplates = templates.some((item) => item.id === template.id)
      ? templates.map((item) => item.id === template.id ? template : item)
      : [template, ...templates];
    persistTemplateLibrary(nextTemplates.slice(0, 12));
    setSelectedTemplateId(template.id);
    const nextRegions = regionsFromTemplate(template);
    setCanvasRegions(nextRegions);
    setSelectedRegionId((current) => nextRegions.find((item) => item.id === current)?.id ?? nextRegions[0]?.id ?? "");
  }

  function buildCurrentTemplate(templateID: string): PaperTemplate {
    const title = canvasTitle.trim() || "未命名试卷模版";
    const questions = canvasRegionsToQuestions(canvasRegions, templateID);
    return {
      id: templateID,
      name: title,
      subject: "数学",
      grade: "六年级",
      questionCount: Math.max(questions.length, 1),
      totalScore: questions.reduce((sum, item) => sum + item.score, 0),
      sourceFileUrl: selectedPaperSource ? `/mock/templates/${selectedPaperSource.id}.pdf` : (selectedTemplate?.sourceFileUrl ?? ""),
      status: selectedTemplate?.id === templateID ? normalizeTemplateStatus(selectedTemplate.status) : "draft",
      version: selectedTemplate?.id === templateID ? (selectedTemplate.version ?? 1) : 1,
      parentId: selectedTemplate?.id === templateID ? (selectedTemplate.parentId ?? "") : "",
      questions
    };
  }

  async function saveCurrentAsTemplate() {
    const localTemplate = buildCurrentTemplate(`tpl_local_${Date.now()}`);
    setTemplatesState({ status: "processing", message: "正在保存模版", detail: "优先写入 Go API 模板库。" });
    try {
      const savedTemplate = await writeTemplateToApi("/api/templates", "POST", localTemplate);
      const nextTemplates = [savedTemplate, ...templates.filter((item) => item.id !== savedTemplate.id)].slice(0, 12);
      persistTemplateLibrary(nextTemplates);
      setSelectedTemplateId(savedTemplate.id);
      setTemplatesState({ status: "success", message: "模版已保存" });
      setNotice(`${savedTemplate.name} 已保存到 API 模版库`);
    } catch {
      persistTemplateLibrary([localTemplate, ...templates].slice(0, 12));
      setSelectedTemplateId(localTemplate.id);
      setTemplatesState({
        status: "error",
        message: "模版保存 API 请求失败",
        detail: "已保存到本地模版库，可稍后重试 API。"
      });
      setNotice(`${localTemplate.name} 已保存到本地模版库`);
    }
  }

  async function updateCurrentTemplate() {
    if (!selectedTemplateId) {
      setNotice("请先选择要更新的模版");
      return;
    }
    if (!canEditSelectedTemplate) {
      setNotice("已发布或停用模版不可直接编辑，请先复制新版本");
      return;
    }
    const localTemplate = buildCurrentTemplate(selectedTemplateId);
    setTemplatesState({ status: "processing", message: "正在更新模版", detail: "优先写入 Go API 模板库。" });
    try {
      const savedTemplate = await writeTemplateToApi(`/api/templates/${encodeURIComponent(selectedTemplateId)}`, "PUT", localTemplate);
      const nextTemplates = templates.map((item) => item.id === savedTemplate.id ? savedTemplate : item);
      persistTemplateLibrary(nextTemplates);
      setTemplatesState({ status: "success", message: "模版已更新" });
      setNotice(`${savedTemplate.name} 已更新到 API 模版库`);
    } catch {
      const nextTemplates = templates.map((item) => item.id === selectedTemplateId ? localTemplate : item);
      persistTemplateLibrary(nextTemplates);
      setTemplatesState({
        status: "error",
        message: "模版更新 API 请求失败",
        detail: "已更新本地模版库，可稍后重试 API。"
      });
      setNotice(`${localTemplate.name} 已更新到本地模版库`);
    }
  }

  function applyTemplateFromLibrary(template: PaperTemplate) {
    const regions = regionsFromTemplate(template);
    setSelectedTemplateId(template.id);
    setCanvasTitle(template.name);
    setCanvasRegions(regions);
    setSelectedRegionId(regions[0]?.id ?? "");
    setNotice(`已引用模版：${template.name}`);
  }

  async function deleteTemplateFromLibrary(templateID: string) {
    const nextTemplates = templates.filter((item) => item.id !== templateID);
    setTemplatesState({ status: "processing", message: "正在删除模版", detail: "优先从 Go API 模板库删除。" });
    try {
      const response = await fetch(`/api/templates/${encodeURIComponent(templateID)}`, { method: "DELETE" });
      if (!response.ok) {
        throw new Error("template delete api failed");
      }
      persistTemplateLibrary(nextTemplates);
      setSelectedTemplateId((current) => current === templateID ? (nextTemplates[0]?.id ?? "") : current);
      setSelectedTemplateIds((current) => current.filter((id) => id !== templateID));
      setTemplatesState(nextTemplates.length > 0 ? { status: "success", message: "模版已删除" } : { status: "empty", message: "暂无可用模版" });
      setNotice("已从 API 模版库删除");
    } catch {
      persistTemplateLibrary(nextTemplates);
      setSelectedTemplateId((current) => current === templateID ? (nextTemplates[0]?.id ?? "") : current);
      setSelectedTemplateIds((current) => current.filter((id) => id !== templateID));
      setTemplatesState({
        status: "error",
        message: "模版删除 API 请求失败",
        detail: "已从本地模版库删除，可稍后同步 API。"
      });
      setNotice("已从本地模版库删除");
    }
  }

  async function copyTemplateFromLibrary(templateID: string) {
    const source = templates.find((item) => item.id === templateID);
    if (!source) {
      setNotice("未找到可复制的模版");
      return;
    }
    setTemplatesState({ status: "processing", message: "正在复制模版", detail: "优先调用 Go API 复制。" });
    try {
      const response = await fetch(`/api/templates/${encodeURIComponent(templateID)}/copy`, { method: "POST" });
      if (!response.ok) {
        throw new Error("template copy api failed");
      }
      const result = await response.json() as TemplateMutationResponse;
      persistTemplateLibrary([result.template, ...templates].slice(0, 12));
      setSelectedTemplateId(result.template.id);
      setTemplatesState({ status: "success", message: "模版已复制" });
      setNotice(`${result.template.name} 已复制到 API 模版库`);
    } catch {
      const copiedTemplate: PaperTemplate = {
        ...source,
        id: `tpl_copy_${Date.now()}`,
        name: `${source.name} 副本`,
        status: "draft",
        version: (source.version ?? 1) + 1,
        parentId: source.parentId || source.id,
        questions: source.questions.map((question) => ({ ...question, id: `q_copy_${Date.now()}_${question.no}` }))
      };
      persistTemplateLibrary([copiedTemplate, ...templates].slice(0, 12));
      setSelectedTemplateId(copiedTemplate.id);
      setTemplatesState({
        status: "error",
        message: "模版复制 API 请求失败",
        detail: "已复制到本地模版库，可稍后同步 API。"
      });
      setNotice(`${copiedTemplate.name} 已复制到本地模版库`);
    }
  }

  async function updateTemplateStatus(templateID: string, status: TemplateStatus) {
    const source = templates.find((item) => item.id === templateID);
    if (!source) {
      setNotice("未找到要流转的模版");
      return;
    }
    const statusLabel = templateStatusLabels[status];
    setTemplatesState({ status: "processing", message: `正在${statusLabel}模版`, detail: "优先写入 Go API 模板库状态。" });
    try {
      const response = await fetch(`/api/templates/${encodeURIComponent(templateID)}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status })
      });
      if (!response.ok) {
        throw new Error("template status api failed");
      }
      const result = await response.json() as TemplateMutationResponse;
      const nextTemplates = templates.map((item) => item.id === result.template.id ? result.template : item);
      persistTemplateLibrary(nextTemplates);
      setTemplatesState({ status: "success", message: `模版已${statusLabel}` });
      setNotice(`${result.template.name} 已${statusLabel}`);
    } catch {
      const localTemplate = { ...source, status };
      const nextTemplates = templates.map((item) => item.id === templateID ? localTemplate : item);
      persistTemplateLibrary(nextTemplates);
      setTemplatesState({
        status: "error",
        message: "模版状态 API 请求失败",
        detail: "已更新本地模版库状态，可稍后同步 API。"
      });
      setNotice(`${source.name} 已在本地标记为${statusLabel}`);
    }
  }

  function saveTemplateDraft() {
    const now = new Date();
    const title = canvasTitle.trim() || "未命名试卷模板";
    const draft: TemplateDraft = {
      id: `draft_${Date.now()}`,
      title,
      sourceTitle: selectedPaperSource?.title ?? "未选择试卷来源",
      updatedAt: `${now.getMonth() + 1}/${now.getDate()} ${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`,
      size: canvasSize,
      zoom: canvasZoom,
      regions: canvasRegions
    };
    persistDrafts([draft, ...templateDrafts.filter((item) => item.title !== title)].slice(0, 8));
    setNotice(`${title} 已保存到草稿箱`);
  }

  function loadTemplateDraft(draft: TemplateDraft) {
    setCanvasTitle(draft.title);
    setCanvasSize(draft.size);
    setCanvasZoom(draft.zoom);
    setCanvasRegions(draft.regions);
    setSelectedRegionId(draft.regions[0]?.id ?? "");
    setSelectedPaperSourceId("");
    setNotice(`已从草稿箱打开：${draft.title}`);
  }

  function regionsFromAIQuestions(questions: QuestionTemplate[]): CanvasRegion[] {
    return regionsFromTemplate({
      id: selectedTemplateId || "ai_suggestion",
      name: canvasTitle,
      subject: "数学",
      grade: "六年级",
      questionCount: questions.length,
      totalScore: questions.reduce((sum, question) => sum + question.score, 0),
      status: "draft",
      version: selectedTemplate?.version ?? 1,
      questions
    });
  }

  async function generateTemplateAISuggestions() {
    if (!canEditSelectedTemplate) {
      setNotice("已发布或停用模板不可直接写入 AI 建议，请先复制新版本");
      return;
    }
    if (!selectedTemplateId) {
      setNotice("请先保存或选择一个草稿模板，再生成 AI 拆卷建议");
      return;
    }
    setAiSuggestionState({ status: "processing", message: "正在生成 AI 拆卷建议", detail: "会识别题区、题型、题号并等待教师确认。" });
    try {
      const response = await fetch(`/api/templates/${encodeURIComponent(selectedTemplateId)}/ai-suggestions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          paperName: canvasTitle,
          sourceFileUrl: selectedPaperSource ? `/mock/templates/${selectedPaperSource.id}.pdf` : (selectedTemplate?.sourceFileUrl ?? "")
        })
      });
      if (!response.ok) {
        throw new Error("template ai suggestion api failed");
      }
      const result = await response.json() as TemplateAISuggestionResponse;
      const nextRegions = regionsFromAIQuestions(result.suggestedQuestions);
      setAiSuggestedRegions(nextRegions);
      setCanvasRegions(nextRegions);
      setSelectedRegionId(nextRegions[0]?.id ?? "");
      setAiSuggestionState({
        status: "success",
        message: `AI 已建议 ${result.questionCount} 个题区`,
        detail: `${result.source} · 总分 ${result.totalScore} · 请确认后写入模板库。`
      });
      setNotice("AI 拆卷建议已载入画布，确认后可写入模板库");
    } catch {
      const fallback = regionsFromAIQuestions([
        { id: `ai_q_${Date.now()}_1`, no: "1", type: "single_choice", score: 2, standardAnswer: "A", scoringRules: ["选对 A 得 2 分"], knowledge: ["分数"], region: { page: 1, x: 120, y: 260, width: 480, height: 80 } },
        { id: `ai_q_${Date.now()}_15`, no: "15", type: "subjective", score: 10, standardAnswer: "列比例关系并计算。", scoringRules: ["建模 2 分", "列式 4 分", "计算 2 分", "答语 2 分"], knowledge: ["比例", "应用题建模"], region: { page: 2, x: 96, y: 420, width: 620, height: 180 } }
      ]);
      setAiSuggestedRegions(fallback);
      setCanvasRegions(fallback);
      setSelectedRegionId(fallback[0]?.id ?? "");
      setAiSuggestionState({
        status: "error",
        message: "AI 拆卷 API 请求失败",
        detail: "已载入本地建议，可确认后写入草稿模板。"
      });
      setNotice("AI 拆卷 API 不可用，已使用本地建议");
    }
  }

  async function confirmTemplateAISuggestions() {
    if (aiSuggestedRegions.length === 0) {
      setNotice("请先生成 AI 拆卷建议");
      return;
    }
    await saveCurrentRegions();
    setAiSuggestedRegions([]);
    setAiSuggestionState({ status: "success", message: "AI 建议题区已写入模板库" });
  }

  function sendGuardianReminders() {
    const names = analytics.studentRisks.map((item) => item.studentName).join("、");
    setNotice(names ? `已生成 ${names} 的家长提醒` : "当前没有需要提醒的学生");
  }

  function toggleSelected(id: string, selectedIds: string[], setter: (value: string[]) => void) {
    setter(selectedIds.includes(id) ? selectedIds.filter((item) => item !== id) : [...selectedIds, id]);
  }

  function runBatchAction(label: string, count: number) {
    setNotice(count > 0 ? `${label}：已选择 ${count} 项` : "请先选择要批量处理的数据");
  }

  function retrySelectedScanTasks() {
    const tasks = dashboard.scanQueue.filter((job) => selectedScanIds.includes(job.id));
    if (tasks.length === 0) {
      setNotice("请先选择要重试的扫描任务");
      return;
    }
    tasks.forEach((task) => void retryScanTask(task));
  }

  async function loadDashboard() {
    setDashboardState((current) => nextLoadingState(current, "正在加载工作台数据", "正在刷新工作台数据"));
    try {
      const response = await fetch("/api/dashboard");
      if (!response.ok) {
        throw new Error("dashboard api failed");
      }
      const data = normalizeDashboardData(await response.json() as Partial<DashboardData> | null);
      setDashboard(data);
      setDashboardState(hasDashboardData(data)
        ? { status: "success", message: "工作台数据已更新" }
        : { status: "empty", message: "暂无工作台任务", detail: "扫描队列、复核队列和统计指标当前为空。" });
      return data;
    } catch {
      setDashboard(fallbackDashboard);
      setDashboardState({
        status: "error",
        message: "工作台 API 请求失败",
        detail: "已展示本地演示数据，可重试连接 Go API。"
      });
      return fallbackDashboard;
    }
  }

  async function loadReviewQueue(silent = false) {
    if (!silent) {
      setSubjectiveState((current) => nextLoadingState(current, "正在加载复核队列", "正在刷新复核队列"));
    }
    try {
      const response = await fetch("/api/grading/subjective/reviews");
      if (!response.ok) {
        throw new Error("review queue api failed");
      }
      const result = await response.json() as ReviewQueueResponse;
      const items = Array.isArray(result.items) ? result.items : [];
      setDashboard((current) => ({ ...current, reviewQueue: items }));
      if (!silent) {
        setSubjectiveState(items.length > 0
          ? { status: "success", message: "复核队列已更新" }
          : { status: "empty", message: "暂无待复核主观题", detail: "扫描阅卷完成后会进入这里。" });
      }
      return items;
    } catch {
      setDashboard((current) => ({ ...current, reviewQueue: fallbackDashboard.reviewQueue }));
      if (!silent) {
        setSubjectiveState({
          status: "error",
          message: "复核队列 API 请求失败",
          detail: "已展示本地演示队列，可稍后重试。"
        });
      }
      return fallbackDashboard.reviewQueue;
    }
  }

  async function loadScanTasks(silent = false) {
    if (!silent) {
      setDashboardState((current) => nextLoadingState(current, "正在加载扫描任务", "正在刷新扫描任务"));
    }
    try {
      const response = await fetch("/api/scan/tasks");
      if (!response.ok) {
        throw new Error("scan tasks api failed");
      }
      const result = await response.json() as ScanTaskListResponse;
      const tasks = Array.isArray(result.tasks) ? result.tasks : [];
      setDashboard((current) => ({
        ...current,
        scanQueue: tasks
      }));
      if (!silent) {
        setDashboardState(tasks.length > 0
          ? { status: "success", message: "扫描任务已更新" }
          : { status: "empty", message: "暂无扫描任务", detail: "上传扫描件并创建任务后会显示在这里。" });
      }
      return tasks;
    } catch {
      if (!silent) {
        setDashboardState({
          status: "error",
          message: "扫描任务 API 请求失败",
          detail: "已保留当前页面任务列表，可稍后重试。"
        });
      }
      return dashboard.scanQueue;
    }
  }

  async function openScanPreview(job: ScanJob) {
    try {
      const response = await fetch(`/api/scan/tasks/${encodeURIComponent(job.id)}/preview`);
      if (!response.ok) {
        throw new Error("scan preview api failed");
      }
      const result = await response.json() as ScanTaskPreviewResponse;
      setScanPreviewTask(result.task);
      setScanPreviewFiles(Array.isArray(result.files) ? result.files : []);
      setScanMatchNames(Object.fromEntries((result.files ?? []).map((file) => [file.key, file.studentName ?? ""])));
      setNotice(`已打开扫描预览：${result.task.title}`);
    } catch {
      const files = job.files ?? [];
      setScanPreviewTask(job);
      setScanPreviewFiles(files);
      setScanMatchNames(Object.fromEntries(files.map((file) => [file.key, file.studentName ?? ""])));
      setNotice("扫描预览 API 不可用，已显示当前任务文件信息");
    }
  }

  async function retryScanTask(job: ScanJob, fileKey = "") {
    setDashboardState({ status: "processing", message: fileKey ? "正在重试单个文件" : "正在重试扫描任务" });
    try {
      const response = await fetch(`/api/scan/tasks/${encodeURIComponent(job.id)}/retry`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ fileKey })
      });
      if (!response.ok) {
        throw new Error("scan retry api failed");
      }
      const result = await response.json() as ScanTaskResponse;
      setDashboard((current) => ({
        ...current,
        scanQueue: current.scanQueue.map((item) => item.id === result.task.id ? result.task : item)
      }));
      if (scanPreviewTask?.id === result.task.id) {
        setScanPreviewTask(result.task);
        setScanPreviewFiles(result.task.files ?? []);
      }
      setDashboardState({ status: "success", message: "扫描任务已重试" });
      setNotice(result.queueError ? `重试已记录，但队列投递失败：${result.queueError}` : "重试任务已重新入队");
    } catch {
      const nextJob = {
        ...job,
        status: "排队中",
        progress: 0,
        retryCount: (job.retryCount ?? 0) + 1,
        failureReason: "",
        queueStatus: "pending",
        files: (job.files ?? []).map((file) => fileKey && file.key !== fileKey ? file : { ...file, status: "待重试", failureReason: "" })
      };
      setDashboard((current) => ({
        ...current,
        scanQueue: current.scanQueue.map((item) => item.id === job.id ? nextJob : item)
      }));
      if (scanPreviewTask?.id === job.id) {
        setScanPreviewTask(nextJob);
        setScanPreviewFiles(nextJob.files ?? []);
      }
      setDashboardState({ status: "error", message: "扫描重试 API 请求失败", detail: "已在本地标记重试，可稍后同步 API。" });
      setNotice("扫描重试 API 不可用，已本地记录重试");
    }
  }

  async function matchScanFile(job: ScanJob, file: ScanUploadFile) {
    const studentName = (scanMatchNames[file.key] ?? "").trim();
    if (!studentName) {
      setNotice("请先填写学生姓名");
      return;
    }
    try {
      const response = await fetch(`/api/scan/tasks/${encodeURIComponent(job.id)}/match`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          fileKey: file.key,
          studentName,
          matchMethod: "manual"
        })
      });
      if (!response.ok) {
        throw new Error("scan match api failed");
      }
      const result = await response.json() as ScanTaskResponse;
      setDashboard((current) => ({
        ...current,
        scanQueue: current.scanQueue.map((item) => item.id === result.task.id ? result.task : item)
      }));
      setScanPreviewTask(result.task);
      setScanPreviewFiles(result.task.files ?? []);
      setNotice(`${file.fileName} 已匹配到 ${studentName}`);
    } catch {
      const nextFiles = scanPreviewFiles.map((item) => item.key === file.key
        ? { ...item, studentName, matchStatus: "matched", matchMethod: "manual" }
        : item);
      setScanPreviewFiles(nextFiles);
      setNotice("学生匹配 API 不可用，已在本地更新预览状态");
    }
  }

  async function loadTemplates() {
    setTemplatesState((current) => nextLoadingState(current, "正在加载模版库", "正在刷新模版库"));
    try {
      const response = await fetch("/api/templates");
      if (!response.ok) {
        throw new Error("templates api failed");
      }
      const data = await response.json() as PaperTemplate[];
      setTemplates(data);
      if (templates.length === 0 && data.length > 0) {
        persistTemplateLibrary(data);
      }
      if (data[0] && !selectedTemplateId) {
        setSelectedTemplateId(data[0].id);
      }
      setTemplatesState(data.length > 0
        ? { status: "success", message: "模版库已更新" }
        : { status: "empty", message: "暂无可用模版", detail: "保存模板或重试接口后会显示在这里。" });
      return data;
    } catch {
      const localTemplates = loadStoredTemplateLibrary();
      setTemplates(localTemplates);
      setTemplatesState({
        status: "error",
        message: "模版库 API 请求失败",
        detail: "已展示本地模版库，可重试连接 Go API。"
      });
      return localTemplates;
    }
  }

  async function loadAnalytics() {
    setAnalyticsState((current) => nextLoadingState(current, "正在加载学情数据", "正在刷新学情数据"));
    try {
      const response = await fetch("/api/analytics/classroom");
      if (!response.ok) {
        throw new Error("analytics api failed");
      }
      const data = normalizeAnalyticsData(await response.json() as Partial<ClassroomAnalytics> | null);
      setAnalytics(data);
      setAnalyticsState(hasAnalyticsData(data)
        ? { status: "success", message: "学情数据已更新" }
        : { status: "empty", message: "暂无学情数据", detail: "当前班级还没有题目、知识点或学生风险统计。" });
      return data;
    } catch {
      setAnalytics(fallbackAnalytics);
      setAnalyticsState({
        status: "error",
        message: "学情 API 请求失败",
        detail: "已展示本地演示数据，可重试连接 Go API。"
      });
      return fallbackAnalytics;
    }
  }

  async function loadWrongQuestions() {
    setMistakesState((current) => nextLoadingState(current, "正在加载错题档案", "正在刷新错题档案"));
    try {
      const response = await fetch("/api/mistakes");
      if (!response.ok) throw new Error("mistakes api failed");
      const data = await response.json() as WrongQuestionListResponse;
      const items = Array.isArray(data.items) ? data.items : [];
      setWrongQuestions(items);
      setSelectedMistake((current) => items.find((item) => item.id === current?.id) ?? items[0] ?? null);
      setMistakesState(items.length > 0
        ? { status: "success", message: "错题档案已更新" }
        : { status: "empty", message: "暂无错题", detail: "阅卷产生失分题后会自动归档。" });
      return items;
    } catch {
      setWrongQuestions(fallbackWrongQuestions);
      setSelectedMistake((current) => fallbackWrongQuestions.find((item) => item.id === current?.id) ?? fallbackWrongQuestions[0]);
      setMistakesState({ status: "error", message: "错题 API 请求失败", detail: "已展示本地演示错题。" });
      return fallbackWrongQuestions;
    }
  }

  async function loadLearningProfile() {
    try {
      const response = await fetch("/api/learning/profile?className=六年级%203%20班");
      if (!response.ok) throw new Error("profile api failed");
      const data = await response.json() as LearningProfile;
      setLearningProfile({
        className: data.className ?? "六年级 3 班",
        knowledgeMastery: Array.isArray(data.knowledgeMastery) ? data.knowledgeMastery : [],
        studentRisks: Array.isArray(data.studentRisks) ? data.studentRisks : [],
        homeworkWatch: Array.isArray(data.homeworkWatch) ? data.homeworkWatch : []
      });
    } catch {
      setLearningProfile(fallbackLearningProfile);
    }
  }

  async function loadGuardianReport(studentName: string) {
    try {
      const response = await fetch("/api/reports/guardian?studentName=" + encodeURIComponent(studentName));
      if (!response.ok) throw new Error("guardian report api failed");
      setGuardianReport(await response.json() as GuardianReport);
      setNotice("已生成 " + studentName + " 的家长报告");
    } catch {
      setGuardianReport({ ...fallbackGuardianReport, studentName });
      setNotice("家长报告 API 不可用，已展示演示报告");
    }
  }

  async function openMistakeDetail(item: WrongQuestion) {
    setSelectedMistake(item);
    try {
      const response = await fetch("/api/mistakes/" + item.id);
      if (response.ok) setSelectedMistake(await response.json() as WrongQuestion);
    } catch {
      setSelectedMistake(item);
    }
  }

  async function createRepracticeTask() {
    const ids = selectedMistakeIds.length > 0
      ? selectedMistakeIds.map(Number)
      : selectedMistake ? [selectedMistake.id] : [];
    if (ids.length === 0) {
      setNotice("请先选择需要再练的错题");
      return;
    }
    try {
      const response = await fetch("/api/mistakes/repractice", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ wrongQuestionIds: ids, title: "错题订正与再练" })
      });
      if (!response.ok) throw new Error("repractice api failed");
      const data = await response.json() as { linkedCount: number };
      setWrongQuestions((current) => current.map((item) => ids.includes(item.id) ? { ...item, repracticeStatus: "assigned" } : item));
      setSelectedMistakeIds([]);
      setNotice("已生成再练任务，关联 " + data.linkedCount + " 道错题");
    } catch {
      setNotice("再练任务创建失败，请检查 Go API 和数据库");
    }
  }

  async function generateScores() {
    setAnalyticsState({ status: "processing", message: "正在统一生成题分和总分" });
    try {
      const response = await fetch(`/api/analytics/generate-scores?className=${encodeURIComponent(analytics.className)}`, {
        method: "POST"
      });
      if (!response.ok) {
        throw new Error("score generation failed");
      }
      const result = await response.json() as ScoreGenerationResponse;
      setNotice(`${result.className} 已生成 ${result.generated} 份成绩`);
      const data = await loadAnalytics();
      setAnalyticsState(hasAnalyticsData(data)
        ? { status: "success", message: "成绩统计已重新生成" }
        : { status: "empty", message: "暂无可统计成绩", detail: "请确认扫描识别和阅卷结果已入库。" });
    } catch {
      setAnalyticsState({
        status: "error",
        message: "成绩生成失败",
        detail: "请确认 Go API 和数据库可用，或稍后重试。"
      });
    }
  }

  function exportScores() {
    setNotice("正在导出成绩单 CSV");
    window.location.href = "/api/analytics/export/scores.csv";
  }

  function confirmObjectiveException(id: number) {
    setAnalytics((current) => ({
      ...current,
      objectiveExceptions: current.objectiveExceptions.map((item) => (
        item.id === id ? { ...item, status: "confirmed" } : item
      ))
    }));
    setNotice("客观题异常已标记为人工确认");
  }

  function questionRegionStyle(region: Region) {
    const pageWidth = 760;
    const pageHeight = 900;
    return {
      height: `${(region.height / pageHeight) * 100}%`,
      left: `${(region.x / pageWidth) * 100}%`,
      top: `${(region.y / pageHeight) * 100}%`,
      width: `${(region.width / pageWidth) * 100}%`
    };
  }

  function canvasRegionStyle(item: CanvasRegion) {
    return {
      backgroundColor: `${item.color}1f`,
      borderColor: item.color,
      borderStyle: item.borderStyle,
      color: item.color,
      height: `${(item.region.height / canvasSize.height) * 100}%`,
      left: `${(item.region.x / canvasSize.width) * 100}%`,
      top: `${(item.region.y / canvasSize.height) * 100}%`,
      width: `${(item.region.width / canvasSize.width) * 100}%`
    };
  }

  function canvasPoint(event: { clientX: number; clientY: number }) {
    const rect = canvasElement?.getBoundingClientRect();
    if (!rect) {
      return { x: 0, y: 0 };
    }
    return {
      x: ((event.clientX - rect.left) / rect.width) * canvasSize.width,
      y: ((event.clientY - rect.top) / rect.height) * canvasSize.height
    };
  }

  function clampRegion(region: Region): Region {
    const minWidth = 54;
    const minHeight = 34;
    const width = Math.max(minWidth, Math.min(region.width, canvasSize.width));
    const height = Math.max(minHeight, Math.min(region.height, canvasSize.height));
    return {
      ...region,
      width,
      height,
      x: Math.max(0, Math.min(region.x, canvasSize.width - width)),
      y: Math.max(0, Math.min(region.y, canvasSize.height - height))
    };
  }

  async function addRegionAt(event: { clientX: number; clientY: number; target: EventTarget; currentTarget: EventTarget }) {
    if (!canEditSelectedTemplate) {
      setNotice("已发布或停用模版不可直接新增题区，请先复制新版本");
      return;
    }
    const point = canvasPoint(event);
    const tool = templateTools[activeTool];
    const nextIndex = canvasRegions.length + 1;
    const nextRegion: CanvasRegion = {
      id: `region_${Date.now()}`,
      no: `${nextIndex}`,
      type: activeTool,
	      label: tool.label,
	      color: tool.color,
	      borderStyle: "solid",
	      score: activeTool === "subjective" ? 10 : 2,
	      standardAnswer: "",
	      scoringRules: [],
	      knowledge: [],
	      region: clampRegion({
        page: 1,
        x: point.x - 90,
        y: point.y - 34,
        width: activeTool === "subjective" ? 220 : 160,
        height: activeTool === "subjective" ? 90 : 64
      })
    };
    setCanvasRegions((current) => [...current, nextRegion]);
    setSelectedRegionId(nextRegion.id);
    setNotice(`已添加${tool.label}区域`);
    if (!selectedTemplateId) {
      return;
    }
    try {
      const result = await writeTemplateRegionToApi(
        `/api/templates/${encodeURIComponent(selectedTemplateId)}/regions`,
        "POST",
        canvasRegionsToQuestions([nextRegion], selectedTemplateId)[0]
      );
      applyTemplateMutation(result.template);
      setSelectedRegionId(result.question.id);
      setTemplatesState({ status: "success", message: "题区已保存" });
      setNotice(`已添加并保存${tool.label}区域`);
    } catch {
      setTemplatesState({
        status: "error",
        message: "题区新增 API 请求失败",
        detail: "已保留本地画布区域，可稍后批量保存题区。"
      });
    }
  }

  function updateSelectedRegion(updater: (region: CanvasRegion) => CanvasRegion) {
    if (!canEditSelectedTemplate) {
      setNotice("已发布或停用模版不可直接编辑，请先复制新版本");
      return;
    }
    setCanvasRegions((current) => current.map((item) => item.id === selectedRegionId ? updater(item) : item));
  }

  function startRegionDrag(
    event: { clientX: number; clientY: number; stopPropagation: () => void; currentTarget: { setPointerCapture?: (pointerId: number) => void }; pointerId: number },
    item: CanvasRegion,
    mode: "move" | "resize"
  ) {
    event.stopPropagation();
    if (!canEditSelectedTemplate) {
      setNotice("已发布或停用模版不可直接拖拽，请先复制新版本");
      return;
    }
    setSelectedRegionId(item.id);
    setDragState({
      id: item.id,
      mode,
      startX: event.clientX,
      startY: event.clientY,
      original: item.region
    });
    event.currentTarget.setPointerCapture?.(event.pointerId);
  }

  function moveRegionDrag(event: { clientX: number; clientY: number }) {
    if (!dragState) {
      return;
    }
    const rect = canvasElement?.getBoundingClientRect();
    if (!rect) {
      return;
    }
    const dx = ((event.clientX - dragState.startX) / rect.width) * canvasSize.width;
    const dy = ((event.clientY - dragState.startY) / rect.height) * canvasSize.height;
    setCanvasRegions((current) => current.map((item) => {
      if (item.id !== dragState.id) {
        return item;
      }
      const nextRegion = dragState.mode === "move"
        ? { ...dragState.original, x: dragState.original.x + dx, y: dragState.original.y + dy }
        : { ...dragState.original, width: dragState.original.width + dx, height: dragState.original.height + dy };
      return { ...item, region: clampRegion(nextRegion) };
    }));
  }

  async function finishRegionDrag() {
    if (!dragState) {
      return;
    }
    const region = canvasRegions.find((item) => item.id === dragState.id);
    setDragState(null);
    if (!region || !selectedTemplateId) {
      return;
    }
    try {
      const result = await writeTemplateRegionToApi(
        `/api/templates/${encodeURIComponent(selectedTemplateId)}/regions/${encodeURIComponent(region.id)}`,
        "PUT",
        canvasRegionsToQuestions([region], selectedTemplateId)[0]
      );
      applyTemplateMutation(result.template);
      setSelectedRegionId(result.question.id);
      setTemplatesState({ status: "success", message: "题区坐标已保存" });
    } catch {
      setTemplatesState({
        status: "error",
        message: "题区坐标 API 请求失败",
        detail: "已保留本地拖拽结果，可稍后批量保存题区。"
      });
    }
  }

  async function saveCurrentRegions() {
    if (!selectedTemplateId) {
      setNotice("请先选择要保存题区的模版");
      return;
    }
    if (!canEditSelectedTemplate) {
      setNotice("已发布或停用模版不可直接保存题区，请先复制新版本");
      return;
    }
    setTemplatesState({ status: "processing", message: "正在保存题区", detail: "批量写入当前画布区域。" });
    try {
      const response = await fetch(`/api/templates/${encodeURIComponent(selectedTemplateId)}/regions`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ questions: canvasRegionsToQuestions(canvasRegions, selectedTemplateId) })
      });
      if (!response.ok) {
        throw new Error("template regions api failed");
      }
      const result = await response.json() as TemplateMutationResponse;
      applyTemplateMutation(result.template);
      setTemplatesState({ status: "success", message: "题区已批量保存" });
      setNotice(`${result.template.name} 的题区已保存到 API`);
    } catch {
      const localTemplate = buildCurrentTemplate(selectedTemplateId);
      const nextTemplates = templates.map((item) => item.id === selectedTemplateId ? localTemplate : item);
      persistTemplateLibrary(nextTemplates);
      setTemplatesState({
        status: "error",
        message: "题区批量保存 API 请求失败",
        detail: "已更新本地模版库，可稍后重试 API。"
      });
      setNotice(`${localTemplate.name} 的题区已保存到本地模版库`);
    }
  }

  async function deleteSelectedRegion() {
    if (!selectedRegionId) {
      return;
    }
    if (!canEditSelectedTemplate) {
      setNotice("已发布或停用模版不可直接删除题区，请先复制新版本");
      return;
    }
    const deletingRegionID = selectedRegionId;
    setCanvasRegions((current) => current.filter((item) => item.id !== selectedRegionId));
    setSelectedRegionId("");
    setNotice("已删除选中区域");
    if (!selectedTemplateId) {
      return;
    }
    try {
      const response = await fetch(
        `/api/templates/${encodeURIComponent(selectedTemplateId)}/regions/${encodeURIComponent(deletingRegionID)}`,
        { method: "DELETE" }
      );
      if (!response.ok) {
        throw new Error("template region delete api failed");
      }
      const result = await response.json() as TemplateMutationResponse;
      applyTemplateMutation(result.template);
      setTemplatesState({ status: "success", message: "题区已删除" });
      setNotice("已从 API 删除选中题区");
    } catch {
      setTemplatesState({
        status: "error",
        message: "题区删除 API 请求失败",
        detail: "已从本地画布删除，可稍后批量保存题区。"
      });
    }
  }

  function applySubjective(data: SubjectiveData) {
    setSubjective(data);
    setSelectedReviewId(data.reviewId);
    setScore(clampScore(data.ai.score, data.fullScore));
    setNote(data.ai.reason);
    setReviewStage("first_review");
    setQueueStatus("已连接数据库队列");
    setSavedState("未保存");
    setPaperZoom(1);
    setPaperRotation(0);
    setPaperOffset({ x: 0, y: 0 });
    void loadGradingHistory(data);
  }

  async function loadGradingHistory(data = subjective) {
    if (!data) {
      setGradingHistory([]);
      return [];
    }
    try {
      const response = await fetch(`/api/grading/subjective/history?submissionId=${encodeURIComponent(data.submissionId)}&questionId=${encodeURIComponent(data.questionId)}`);
      if (!response.ok) {
        throw new Error("grading history api failed");
      }
      const result = await response.json() as GradingHistoryResponse;
      const items = Array.isArray(result.items) ? result.items : [];
      setGradingHistory(items);
      return items;
    } catch {
      const fallbackItems: GradingHistoryItem[] = [
        {
          id: 0,
          submissionId: data.submissionId,
          questionId: data.questionId,
          action: "ai_suggested",
          score: data.ai.score,
          note: data.ai.reason,
          actorName: "AI Worker",
          reviewStage: "ai",
          modelVersion: data.ai.modelVersion ?? "mock-ai-worker-v1",
          createdAt: new Date().toISOString()
        }
      ];
      setGradingHistory(fallbackItems);
      return fallbackItems;
    }
  }

  async function loadSubjective(reviewId?: string) {
    const endpoint = reviewId
      ? `/api/grading/subjective/reviews/${encodeURIComponent(reviewId)}`
      : "/api/grading/subjective/current";
    setIsReviewLoading(true);
    setSubjectiveState((current) => nextLoadingState(current, "正在加载主观题复核项", "正在切换复核项"));
    try {
      const response = await fetch(endpoint);
      if (response.status === 404) {
        setSubjective(null);
        setSelectedReviewId("");
        setQueueStatus("当前没有待复核主观题");
        setSavedState("队列已清空");
        setSubjectiveState({
          status: "empty",
          message: "暂无待复核主观题",
          detail: "队列清空后可刷新工作台查看最新任务。"
        });
        return null;
      }
      if (!response.ok) {
        throw new Error("subjective api failed");
      }
      const data = await response.json() as SubjectiveData;
      applySubjective(data);
      setSubjectiveState({ status: "success", message: "复核项已加载" });
      return data;
    } catch {
      applySubjective(fallbackSubjective);
      setQueueStatus("API 未连接，显示本地示例");
      setSubjectiveState({
        status: "error",
        message: "阅卷队列 API 请求失败",
        detail: "已展示本地主观题示例，可重试连接 Go API。"
      });
      return fallbackSubjective;
    } finally {
      setIsReviewLoading(false);
    }
  }

  async function openReview(item: ReviewItem) {
    setSavedState("加载中");
    openView("grading");
    await loadSubjective(item.id);
  }

  function updateScore(nextScore: number) {
    if (!subjective) {
      return;
    }
    setScore(clampScore(nextScore, subjective.fullScore));
  }

  function acceptAIScore() {
    if (!subjective) {
      return;
    }
    setScore(clampScore(subjective.ai.score, subjective.fullScore));
    setNote(subjective.ai.reason);
    setSavedState("已套用 AI 建议，尚未保存");
  }

  async function openAdjacentReview(direction: 1 | -1) {
    if (filteredReviewQueue.length === 0) {
      setNotice("当前筛选条件下没有复核项");
      return;
    }
    const index = selectedReviewIndex >= 0 ? selectedReviewIndex : 0;
    const next = filteredReviewQueue[index + direction];
    if (!next) {
      setNotice(direction > 0 ? "已经是当前队列最后一题" : "已经是当前队列第一题");
      return;
    }
    await openReview(next);
  }

  function startPaperDrag(event: { clientX: number; clientY: number }) {
    setPaperDragStart({ x: event.clientX, y: event.clientY, ox: paperOffset.x, oy: paperOffset.y });
  }

  function movePaper(event: { clientX: number; clientY: number }) {
    if (!paperDragStart) {
      return;
    }
    setPaperOffset({
      x: paperDragStart.ox + event.clientX - paperDragStart.x,
      y: paperDragStart.oy + event.clientY - paperDragStart.y
    });
  }

  function stopPaperDrag() {
    setPaperDragStart(null);
  }

  async function saveDecision(decision: "accepted_ai" | "modified" | "rejected" | "second_review" | "arbitration" | "spot_check") {
    if (!subjective) {
      setSavedState("没有可保存的复核项");
      return;
    }
    if (scoreError) {
      setSavedState(scoreError);
      return;
    }
    setSavedState("保存中");
    setSubjectiveState({ status: "processing", message: "正在保存教师裁定", detail: "保存成功后会自动加载下一题。" });
    try {
      const response = await fetch("/api/grading/subjective/decision", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          submissionId: subjective.submissionId,
          questionId: subjective.questionId,
          finalScore: score,
          decision,
          teacherNote: note,
          actorName: "陈老师",
          reviewStage,
          modelVersion: subjective.ai.modelVersion ?? "mock-ai-worker-v1"
        })
      });
      if (!response.ok) {
        throw new Error("decision api failed");
      }
      const result = await response.json() as GradingDecisionResponse;
      const nextDashboard = await loadDashboard();
      await loadReviewQueue(true);
      if (result.nextReview) {
        applySubjective(result.nextReview);
        setSubjectiveState({ status: "success", message: "裁定已保存，下一题已加载" });
        setSavedState("已保存，下一题已准备");
        return;
      }
      if (nextDashboard.reviewQueue.length > 0) {
        await loadSubjective(nextDashboard.reviewQueue[0].id);
        setSavedState("已保存，下一题已准备");
        return;
      }
      setSubjective(null);
      setSelectedReviewId("");
      setQueueStatus("当前没有待复核主观题");
      setSavedState("已保存，队列已清空");
      setSubjectiveState({
        status: "empty",
        message: "主观题复核已完成",
        detail: "当前队列没有待处理题目。"
      });
    } catch {
      setGradingHistory((current) => [
        {
          id: Date.now(),
          submissionId: subjective.submissionId,
          questionId: subjective.questionId,
          action: decision,
          score,
          note,
          actorName: "陈老师",
          reviewStage,
          modelVersion: subjective.ai.modelVersion ?? "mock-ai-worker-v1",
          createdAt: new Date().toISOString()
        },
        ...current
      ]);
      setSavedState("本地已记录，API 未连接");
      setSubjectiveState({
        status: "error",
        message: "教师裁定保存失败",
        detail: "请检查 API 连接后重试保存。"
      });
    }
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">C</div>
          <div>
            <strong>Club</strong>
            <span>阅卷与学情平台</span>
          </div>
        </div>

        <nav className="nav">
          {visibleNavItems.map((item) => {
            const Icon = item.icon;
            return (
              <button className={activeView === item.view ? "active" : ""} key={item.view} onClick={() => openView(item.view)} type="button">
                <Icon size={18} />{item.label}
              </button>
            );
          })}
        </nav>

        <div className="role-card">
          <div>
            <span>当前角色</span>
            <strong>{activeRole.label}</strong>
            <small>{activeRole.description}</small>
          </div>
          <ShieldCheck size={18} />
        </div>

        <div className="sidebar-note">
          <Sparkles size={18} />
          <span>AI 只提供建议，教师保留最终评分权。</span>
        </div>
      </aside>

      <main className="main">
        <header className="topbar">
          <div>
            <p className="eyebrow">{currentRole === "student" ? "Student Portal" : currentRole === "guardian" ? "Guardian Portal" : viewCopy[activeView].eyebrow}</p>
            <h1>{currentRole === "student" ? "我的学习进展" : currentRole === "guardian" ? "孩子学情与成长建议" : viewCopy[activeView].title}</h1>
            {activeView !== "templates" ? <span className="top-notice">{notice}</span> : null}
          </div>
          <div className="top-actions">
            <label className="role-switcher">
              角色
              <select onChange={(event: { target: { value: string } }) => switchRole(event.target.value as UserRole)} value={currentRole}>
                {Object.entries(roleConfig).map(([role, config]) => (
                  <option key={role} value={role}>{config.label}</option>
                ))}
              </select>
            </label>
            <button className="icon-button" onClick={() => toggleOverlay("filter")} title="筛选" type="button"><SlidersHorizontal size={18} /></button>
            <button className="icon-button" onClick={() => toggleOverlay("notifications")} title="通知" type="button"><Bell size={18} /></button>
            {can("template:edit") ? (
              <button className="secondary-button" onClick={openTemplateCreate} type="button">
                <FileStack size={18} />新建模板
              </button>
            ) : null}
            {can("scan:create") ? (
              <button
                className="primary-button"
                onClick={activeView === "templates" ? importTemplateScanFromCurrentTask : openScanImport}
                type="button"
              >
                <ScanLine size={18} />导入扫描件
              </button>
            ) : null}
          </div>
          {overlay ? (
            <div className="floating-panel">
              {overlay === "filter" ? (
                <>
                  <p className="eyebrow">Filters</p>
                  <h3>工作台筛选</h3>
                  <button className="template-chip active" onClick={() => applyWorkspaceFilter("class")} type="button">六年级 3 班</button>
                  <button className="template-chip" onClick={() => applyWorkspaceFilter("today")} type="button">今日任务</button>
                  <button className="template-chip" onClick={() => applyWorkspaceFilter("review")} type="button">主观题优先</button>
                </>
              ) : (
                <>
                  <p className="eyebrow">Notifications</p>
                  <h3>待处理提醒</h3>
                  <button className="notice-row notice-action" onClick={openReviewQueue} type="button">主观题待复核：{dashboard.reviewQueue.length} 条</button>
                  <button className="notice-row notice-action" onClick={openScanImport} type="button">扫描队列：{dashboard.scanQueue.length} 个任务</button>
                  <button className="notice-row notice-action" onClick={openAnalyticsView} type="button">学生风险：{analytics.studentRisks.length} 条</button>
                </>
              )}
            </div>
          ) : null}
        </header>

        <ConnectionStatusBar
          apiStatus={activeConnection.apiStatus}
          databaseStatus={activeConnection.databaseStatus}
          note={activeConnection.note}
          storageStatus={activeConnection.storageStatus}
        />

        {(currentRole === "student" || currentRole === "guardian") && activeView === "workspace" ? (
          <PortalView role={currentRole} onNotice={setNotice} />
        ) : null}

        {currentRole === "admin" && activeView === "organization" ? <OrganizationView onNotice={setNotice} /> : null}

        {currentRole !== "student" && currentRole !== "guardian" && activeView !== "templates" && activeView !== "organization" ? (
          <>
            <RequestStateView state={dashboardState} onRetry={() => loadDashboard()} compact />
            {dashboard.metrics.length > 0 ? (
              <section className="metrics-grid">
                {dashboard.metrics.map((metric) => (
                  <button className={`metric metric-${metric.tone}`} key={metric.label} onClick={() => openMetricTarget(metric.label)} type="button">
                    <span>{metric.label}</span>
                    <strong>{metric.value}</strong>
                    <small>{metric.delta}</small>
                  </button>
                ))}
              </section>
            ) : null}
          </>
        ) : null}

        {activeView === "scan" ? (
          <section className="panel view-panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Scan Import</p>
                <h2>创建扫描导入任务</h2>
              </div>
              <button className="ghost-button" onClick={() => openView("workspace")} type="button">返回工作台</button>
            </div>
            <div className="form-grid">
              <label>
                考试/作业名称
                <input onChange={(event: { target: { value: string } }) => setScanTitle(event.target.value)} value={scanTitle} />
              </label>
              <label>
                班级
                <input onChange={(event: { target: { value: string } }) => setScanClassName(event.target.value)} value={scanClassName} />
              </label>
              <label>
                绑定模板
                <select onChange={(event: { target: { value: string } }) => setScanTemplateId(event.target.value)} value={scanTemplateId}>
                  <option value="">请选择模板</option>
                  {publishedTemplates.map((template) => (
                    <option key={template.id} value={template.id}>
                      {template.name} · V{template.version ?? 1}
                    </option>
                  ))}
                </select>
                {publishedTemplates.length === 0 ? <span className="form-hint">请先在模板库发布模板</span> : null}
              </label>
              <label>
                页数
                <input min={1} onChange={(event: { target: { value: string } }) => setScanPages(Number(event.target.value))} type="number" value={scanPages} />
              </label>
              <label className="wide-field">
                备注
                <textarea onChange={(event: { target: { value: string } }) => setScanNotes(event.target.value)} rows={3} value={scanNotes} />
              </label>
            </div>
            <div className="upload-zone">
              <ScanLine size={28} />
              <strong>扫描件暂存区</strong>
              <span>支持 PDF、PNG、JPG、WebP 和 ZIP 扫描包，单文件不超过 {formatFileSize(maxScanFileSizeBytes)}。</span>
              <input
                accept={allowedScanExtensions.join(",")}
                multiple
                onChange={(event: { target: { files: FileList | null } }) => selectScanFiles(event.target.files)}
                type="file"
              />
              {scanFileError ? <span className="form-error">{scanFileError}</span> : null}
              {scanFiles.length > 0 ? (
                <div className="selected-files">
                  {scanFiles.map((file) => (
                    <span key={`${file.name}-${file.size}`}>{file.name} · {formatFileSize(file.size)}</span>
                  ))}
                </div>
              ) : null}
              {scanUploadedFiles.length > 0 ? (
                <div className="selected-files uploaded">
                  {scanUploadedFiles.map((file) => (
                    <span key={file.key}>{file.fileName} · {file.key}</span>
                  ))}
                </div>
              ) : null}
              <div className="top-actions">
                <button className="primary-button" onClick={submitScanImport} type="button">开始导入</button>
              </div>
            </div>
            <TableToolbar
              batchLabel="批量重试"
              filterOptions={[
                { label: "全部状态", value: "all" },
                { label: "识别中", value: "识别" },
                { label: "等待", value: "等待" },
                { label: "待导入", value: "待导入" },
                { label: "待 OCR", value: "待 OCR" }
              ]}
              filterValue={scanFilter}
              onBatchAction={retrySelectedScanTasks}
              onFilterChange={(value) => {
                setScanFilter(value);
                setScanPage(1);
              }}
              onSearchChange={(value) => {
                setScanSearch(value);
                setScanPage(1);
              }}
              onSortChange={(value) => {
                setScanSort(value);
                setScanPage(1);
              }}
              searchPlaceholder="试卷、班级或状态"
              searchValue={scanSearch}
              selectedCount={selectedScanIds.length}
              sortOptions={[
                { label: "进度高到低", value: "progress_desc" },
                { label: "进度低到高", value: "progress_asc" },
                { label: "页数多到少", value: "pages_desc" }
              ]}
              sortValue={scanSort}
              totalCount={filteredScanQueue.length}
            />
            <div className="scan-list table-list">
              {filteredScanQueue.length > 0 ? (
                pagedScanQueue.map((job) => (
                  <div className="scan-row table-row" key={job.id}>
                    <input
                      aria-label={`选择${job.title}`}
                      checked={selectedScanIds.includes(job.id)}
                      onChange={() => toggleSelected(job.id, selectedScanIds, setSelectedScanIds)}
                      type="checkbox"
                    />
                    <div>
                      <strong>{job.title}</strong>
                      <span>
                        {job.className} · {job.pages} 页 · {job.status} · {scanQueueLabel(job.queueStatus)}
                        {job.templateId ? ` · 模板 ${job.templateId} V${job.templateVersion ?? 1}` : ""} · 任务 {job.id}
                      </span>
                      {job.failureReason ? <small className="row-error">{job.failureReason}</small> : null}
                    </div>
                    <div className="progress-wrap" aria-label={`${job.progress}%`}>
                      <div className="progress-track">
                        <div className="progress-fill" style={{ width: `${job.progress}%` }} />
                      </div>
                      <em>{job.progress}%</em>
                    </div>
                    <div className="row-actions">
                      <button className="template-chip" onClick={() => void openScanPreview(job)} type="button">预览</button>
                      <button className="template-chip" onClick={() => void retryScanTask(job)} type="button">重试</button>
                    </div>
                  </div>
                ))
              ) : (
                <RequestStateView
                  compact
                  onRetry={() => loadDashboard()}
                  state={{ status: "empty", message: "没有符合条件的扫描任务", detail: "调整搜索或筛选条件后重试。" }}
                />
              )}
            </div>
            <TablePagination page={scanPage} total={filteredScanQueue.length} onPageChange={setScanPage} />
            {scanPreviewTask ? (
              <div className="scan-preview">
                <div className="panel-head">
                  <div>
                    <p className="eyebrow">Scan Preview</p>
                    <h3>{scanPreviewTask.title} · 文件预览</h3>
                  </div>
                  <button className="ghost-button" onClick={() => setScanPreviewTask(null)} type="button">关闭预览</button>
                </div>
                {scanPreviewFiles.length > 0 ? (
                  <div className="preview-file-list">
                    {scanPreviewFiles.map((file) => (
                      <div className="preview-file-row" key={file.key}>
                        <div>
                          <strong>{file.fileName}</strong>
                          <span>第 {file.page ?? "-"} 页 · {file.status ?? "uploaded"} · {scanMatchLabel(file.matchStatus)}</span>
                          {file.failureReason ? <small className="row-error">{file.failureReason}</small> : null}
                        </div>
                        <a className="template-chip" href={file.url || "#"} rel="noreferrer" target="_blank">查看原件</a>
                        <label>
                          匹配学生
                          <input
                            onChange={(event: { target: { value: string } }) => setScanMatchNames((current) => ({ ...current, [file.key]: event.target.value }))}
                            value={scanMatchNames[file.key] ?? file.studentName ?? ""}
                          />
                        </label>
                        <button className="template-chip active" onClick={() => void matchScanFile(scanPreviewTask, file)} type="button">确认匹配</button>
                        <button className="template-chip" onClick={() => void retryScanTask(scanPreviewTask, file.key)} type="button">重试文件</button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <RequestStateView
                    compact
                    state={{ status: "empty", message: "暂无扫描文件", detail: "任务文件上传完成后会在这里预览原件和匹配结果。" }}
                  />
                )}
              </div>
            ) : null}
          </section>
        ) : null}

        {(activeView === "workspace" || activeView === "grading") && (can("scan:create") || can("grading:review")) ? (
        <section className="work-grid">
          {can("scan:create") && activeView === "workspace" ? (
          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Scan Queue</p>
                <h2>扫描处理队列</h2>
              </div>
              <button className="ghost-button" onClick={openScanImport} type="button">查看全部</button>
            </div>
            <TableToolbar
              batchLabel="批量重试"
              filterOptions={[
                { label: "全部状态", value: "all" },
                { label: "识别中", value: "识别" },
                { label: "等待", value: "等待" },
                { label: "待导入", value: "待导入" },
                { label: "待 OCR", value: "待 OCR" }
              ]}
              filterValue={scanFilter}
              onBatchAction={retrySelectedScanTasks}
              onFilterChange={(value) => {
                setScanFilter(value);
                setScanPage(1);
              }}
              onSearchChange={(value) => {
                setScanSearch(value);
                setScanPage(1);
              }}
              onSortChange={(value) => {
                setScanSort(value);
                setScanPage(1);
              }}
              searchPlaceholder="试卷、班级或状态"
              searchValue={scanSearch}
              selectedCount={selectedScanIds.length}
              sortOptions={[
                { label: "进度高到低", value: "progress_desc" },
                { label: "进度低到高", value: "progress_asc" },
                { label: "页数多到少", value: "pages_desc" }
              ]}
              sortValue={scanSort}
              totalCount={filteredScanQueue.length}
            />
            <div className="scan-list table-list">
              {filteredScanQueue.length > 0 ? (
                pagedScanQueue.map((job) => (
                  <div className="scan-row table-row" key={job.id}>
                    <input
                      aria-label={`选择${job.title}`}
                      checked={selectedScanIds.includes(job.id)}
                      onChange={() => toggleSelected(job.id, selectedScanIds, setSelectedScanIds)}
                      type="checkbox"
                    />
                    <div>
                      <strong>{job.title}</strong>
                      <span>
                        {job.className} · {job.pages} 页 · {job.status} · {scanQueueLabel(job.queueStatus)}
                        {job.templateId ? ` · 模板 ${job.templateId} V${job.templateVersion ?? 1}` : ""} · 任务 {job.id}
                      </span>
                      {job.failureReason ? <small className="row-error">{job.failureReason}</small> : null}
                    </div>
                    <div className="progress-wrap" aria-label={`${job.progress}%`}>
                      <div className="progress-track">
                        <div className="progress-fill" style={{ width: `${job.progress}%` }} />
                      </div>
                      <em>{job.progress}%</em>
                    </div>
                    <div className="row-actions">
                      <button className="template-chip" onClick={() => void openScanPreview(job)} type="button">预览</button>
                      <button className="template-chip" onClick={() => void retryScanTask(job)} type="button">重试</button>
                    </div>
                  </div>
                ))
              ) : (
                <RequestStateView
                  compact
                  onRetry={() => loadDashboard()}
                  state={{ status: "empty", message: "没有符合条件的扫描任务", detail: "调整搜索或筛选条件后重试。" }}
                />
              )}
            </div>
            <TablePagination page={scanPage} total={filteredScanQueue.length} onPageChange={setScanPage} />
          </div>
          ) : null}

          {can("grading:review") ? (
          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Review Queue</p>
                <h2>主观题复核</h2>
              </div>
              <button className="ghost-button" onClick={openReviewQueue} type="button"><PenLine size={16} />进入阅卷</button>
            </div>
            <TableToolbar
              batchLabel="批量分配"
              filterOptions={[
                { label: "全部置信度", value: "all" },
                { label: "高置信度", value: "high" },
                { label: "需重点看", value: "low" }
              ]}
              filterValue={reviewFilter}
              onBatchAction={() => runBatchAction("复核队列批量操作", selectedReviewIds.length)}
              onFilterChange={(value) => {
                setReviewFilter(value);
                setReviewPage(1);
              }}
              onSearchChange={(value) => {
                setReviewSearch(value);
                setReviewPage(1);
              }}
              onSortChange={(value) => {
                setReviewSort(value);
                setReviewPage(1);
              }}
              searchPlaceholder="学生、试卷或题号"
              searchValue={reviewSearch}
              selectedCount={selectedReviewIds.length}
              sortOptions={[
                { label: "置信度高到低", value: "confidence_desc" },
                { label: "题号升序", value: "question_asc" },
                { label: "学生姓名", value: "student_asc" }
              ]}
              sortValue={reviewSort}
              totalCount={filteredReviewQueue.length}
            />
            <div className="review-filter-grid">
              <label>
                班级
                <select onChange={(event: { target: { value: string } }) => { setReviewClassFilter(event.target.value); setReviewPage(1); }} value={reviewClassFilter}>
                  <option value="all">全部班级</option>
                  {reviewClassOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                </select>
              </label>
              <label>
                试卷
                <select onChange={(event: { target: { value: string } }) => { setReviewPaperFilter(event.target.value); setReviewPage(1); }} value={reviewPaperFilter}>
                  <option value="all">全部试卷</option>
                  {reviewPaperOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                </select>
              </label>
              <label>
                题号
                <select onChange={(event: { target: { value: string } }) => { setReviewQuestionFilter(event.target.value); setReviewPage(1); }} value={reviewQuestionFilter}>
                  <option value="all">全部题号</option>
                  {reviewQuestionOptions.map((option) => <option key={option.value} value={option.value}>第 {option.label} 题</option>)}
                </select>
              </label>
              <label>
                状态
                <select onChange={(event: { target: { value: string } }) => { setReviewStatusFilter(event.target.value); setReviewPage(1); }} value={reviewStatusFilter}>
                  <option value="all">全部状态</option>
                  <option value="pending">待复核</option>
                  <option value="second_review">二审中</option>
                  <option value="arbitration">仲裁中</option>
                </select>
              </label>
            </div>
            <div className="review-list table-list">
              {filteredReviewQueue.length > 0 ? (
                pagedReviewQueue.map((item) => (
                  <div className={item.id === selectedReview?.id ? "review-row table-row active" : "review-row table-row"} key={item.id}>
                    <input
                      aria-label={`选择${item.studentName}第${item.questionNo}题`}
                      checked={selectedReviewIds.includes(item.id)}
                      onChange={() => toggleSelected(item.id, selectedReviewIds, setSelectedReviewIds)}
                      type="checkbox"
                    />
                    <button className="review-row-main" onClick={() => openReview(item)} type="button">
                      <span>{item.studentName}</span>
                      <strong>第 {item.questionNo} 题</strong>
                      <em>{item.aiAdvice}</em>
                      <small>{item.className ?? "未关联班级"} · {reviewStatusLabel(item.status)} · {reviewStageLabel(item.reviewStage)} · 置信度 {item.confidence}%</small>
                    </button>
                  </div>
                ))
              ) : (
                <RequestStateView
                  compact
                  onRetry={() => loadDashboard()}
                  state={{ status: "empty", message: "没有符合条件的复核项", detail: "调整搜索或筛选条件后重试。" }}
                />
              )}
            </div>
            <TablePagination page={reviewPage} total={filteredReviewQueue.length} onPageChange={setReviewPage} />
          </div>
          ) : null}
        </section>
        ) : null}

        {can("grading:review") && (activeView === "workspace" || activeView === "grading") ? (
        <section className="grading-panel">
          {subjective ? (
            <>
              <div className="grading-head">
                <div>
                  <p className="eyebrow">Subjective Grading</p>
                  <h2>{subjective.paperName} · 第 {subjective.questionNo} 题</h2>
                  <span>{subjective.className} · {subjective.studentName} · {queueStatus}</span>
                </div>
                <div className="grading-head-actions">
                  <div className="segmented">
                    <button className={activeMode === "review" ? "active" : ""} onClick={() => setActiveMode("review")}>左右分屏批阅</button>
                    <button className={activeMode === "template" ? "active" : ""} onClick={() => setActiveMode("template")}>模板信息</button>
                  </div>
                  <div className="review-nav-actions">
                    <button className="ghost-button" disabled={selectedReviewIndex <= 0} onClick={() => void openAdjacentReview(-1)} type="button">上一题</button>
                    <button className="ghost-button" onClick={() => void openAdjacentReview(1)} type="button"><StepForward size={16} />跳过</button>
                    <button className="secondary-button" disabled={selectedReviewIndex < 0 || selectedReviewIndex >= filteredReviewQueue.length - 1} onClick={() => void openAdjacentReview(1)} type="button">下一题</button>
                  </div>
                </div>
              </div>
              <RequestStateView state={subjectiveState} onRetry={() => loadSubjective(selectedReviewId)} compact />

              {activeMode === "review" ? (
                <div className={isReviewLoading ? "split-review loading" : "split-review"}>
                  <article className="answer-pane standard">
                    <div className="pane-title">
                      <Check size={18} />
                      <h3>标准答案与评分规则</h3>
                    </div>
                    <p className="answer-copy">{subjective.standardAnswer.content}</p>
                    <div className="rule-list">
                      {subjective.standardAnswer.scoringRules.map((rule) => (
                        <label className="rule-row" key={rule}>
                          <input defaultChecked type="checkbox" />
                          <span>{rule}</span>
                        </label>
                      ))}
                    </div>
                    <div className="tag-row">
                      {subjective.standardAnswer.knowledge.map((tag) => (
                        <span key={tag}>{tag}</span>
                      ))}
                    </div>
                  </article>

                  <article className="answer-pane student">
                    <div className="pane-title">
                      <MessageSquareText size={18} />
                      <h3>学生答案与 AI 建议</h3>
                    </div>
                    <div className="paper-tools" aria-label="试卷图像工具">
                      <button className="tool-button" onClick={() => setPaperZoom((value) => Math.max(0.6, Number((value - 0.1).toFixed(1))))} type="button"><ZoomOut size={15} />缩小</button>
                      <span>{Math.round(paperZoom * 100)}%</span>
                      <button className="tool-button" onClick={() => setPaperZoom((value) => Math.min(1.8, Number((value + 0.1).toFixed(1))))} type="button"><ZoomIn size={15} />放大</button>
                      <button className="tool-button" onClick={() => setPaperRotation((value) => (value + 90) % 360)} type="button"><RotateCcw size={15} />旋转</button>
                      <button className="tool-button" onClick={() => { setPaperZoom(1); setPaperRotation(0); setPaperOffset({ x: 0, y: 0 }); }} type="button"><TimerReset size={15} />复位</button>
                    </div>
                    <div
                      className={paperDragStart ? "student-paper dragging" : "student-paper"}
                      onMouseDown={startPaperDrag}
                      onMouseLeave={stopPaperDrag}
                      onMouseMove={movePaper}
                      onMouseUp={stopPaperDrag}
                    >
                      <div
                        className="student-paper-page"
                        style={{ transform: `translate(${paperOffset.x}px, ${paperOffset.y}px) scale(${paperZoom}) rotate(${paperRotation}deg)` }}
                      >
                        <div className="paper-line wide" />
                        <div className="paper-line medium" />
                        <p>{subjective.studentAnswer.ocrText}</p>
                        <div className="answer-highlight">第 {subjective.questionNo} 题作答区</div>
                        <div className="paper-line short" />
                      </div>
                    </div>
                    <div className="ai-box">
                      <div>
                        <span>AI 建议分</span>
                        <strong>{subjective.ai.score} / {subjective.fullScore}</strong>
                      </div>
                      <em>置信度 {subjective.ai.confidence}%</em>
                      <p>{subjective.ai.reason}</p>
                      <ul>
                        {subjective.ai.comments.map((comment) => (
                          <li key={comment}>{comment}</li>
                        ))}
                      </ul>
                    </div>
                  </article>

                  <aside className="score-panel">
                    <h3>教师裁定</h3>
                    <label>
                      最终得分
                      <input
                        max={subjective.fullScore}
                        min={0}
                        onChange={(event: { target: { value: string } }) => setScore(Number(event.target.value))}
                        step={0.5}
                        type="number"
                        value={score}
                      />
                    </label>
                    <div className="score-stepper">
                      <button className="tool-button" onClick={() => updateScore(score - 0.5)} type="button">-0.5</button>
                      <button className="tool-button" onClick={() => updateScore(0)} type="button">零分</button>
                      <button className="tool-button" onClick={acceptAIScore} type="button">AI 分</button>
                      <button className="tool-button" onClick={() => updateScore(subjective.fullScore)} type="button">满分</button>
                      <button className="tool-button" onClick={() => updateScore(score + 0.5)} type="button">+0.5</button>
                    </div>
                    {scoreError ? <span className="form-error">{scoreError}</span> : null}
                    <label>
                      复核阶段
                      <select onChange={(event: { target: { value: string } }) => setReviewStage(event.target.value)} value={reviewStage}>
                        <option value="first_review">一审</option>
                        <option value="second_review">二审</option>
                        <option value="spot_check">抽检</option>
                        <option value="arbitration">仲裁</option>
                      </select>
                    </label>
                    <label>
                      批注
                      <textarea onChange={(event: { target: { value: string } }) => setNote(event.target.value)} value={note} />
                    </label>
                    {can("grading:decide") ? (
                      <div className="decision-actions">
                        <button className="primary-button" disabled={Boolean(scoreError)} onClick={() => saveDecision("accepted_ai")}><Check size={18} />接受 AI</button>
                        <button className="secondary-button" disabled={Boolean(scoreError)} onClick={() => saveDecision("modified")}><PenLine size={18} />修改保存</button>
                        <button className="ghost-button" disabled={Boolean(scoreError)} onClick={() => saveDecision("rejected")}>驳回建议</button>
                        <button className="ghost-button" disabled={Boolean(scoreError)} onClick={() => saveDecision("second_review")}>提交二审</button>
                        <button className="ghost-button" disabled={Boolean(scoreError)} onClick={() => saveDecision("arbitration")}>提交仲裁</button>
                      </div>
                    ) : (
                      <div className="permission-note">当前角色仅可查看复核内容，不能保存教师裁定。</div>
                    )}
                    <div className="shortcut-grid">
                      <span>A 接受 AI</span>
                      <span>M 保存</span>
                      <span>R 驳回</span>
                      <span>F 满分</span>
                      <span>Z 零分</span>
                      <span>N 下一题</span>
                    </div>
                    <span className="save-state">{savedState}</span>
                  </aside>
                </div>
              ) : (
                <div className="template-preview">
                  <div className="paper-canvas">
                    <div className="paper-title-line">{selectedTemplate?.name ?? "试卷模板"}</div>
                    {selectedTemplate?.questions.map((question) => (
                      <div
                        className={question.id === selectedTemplateQuestion?.id ? "question-region active" : "question-region"}
                        key={question.id}
                        style={questionRegionStyle(question.region)}
                      >
                        {question.no} · {question.type === "subjective" ? "主观题" : "客观题"}
                      </div>
                    ))}
                  </div>
                  <div className="template-side">
                    <h3>{selectedTemplate?.name ?? "试卷模板"}</h3>
                    <div className="template-selector">
                      {templates.map((template) => (
                        <button
                          className={template.id === selectedTemplate?.id ? "template-chip active" : "template-chip"}
                          key={template.id}
                          onClick={() => setSelectedTemplateId(template.id)}
                          type="button"
                        >
                          {template.grade} · {template.subject}
                        </button>
                      ))}
                    </div>
                    <p>{selectedTemplate?.questionCount ?? 0} 题 · 总分 {selectedTemplate?.totalScore ?? 0} 分</p>
                    <p>
                      当前区域：第 {selectedTemplateQuestion?.no ?? subjective.questionNo} 题 ·
                      满分 {selectedTemplateQuestion?.score ?? subjective.fullScore} 分
                    </p>
                    <p>知识点：{(selectedTemplateQuestion?.knowledge ?? subjective.standardAnswer.knowledge).join("、")}</p>
                    {can("template:edit") ? (
                      <button className="secondary-button" onClick={() => openView("templates")} type="button"><FileStack size={18} />编辑模板</button>
                    ) : null}
                  </div>
                </div>
              )}
              {activeMode === "review" ? (
                <section className="grading-history">
                  <div className="panel-head">
                    <div>
                      <p className="eyebrow">Audit Trail</p>
                      <h3>批阅历史</h3>
                    </div>
                    <button className="ghost-button" onClick={() => void loadGradingHistory()} type="button"><RefreshCw size={16} />刷新</button>
                  </div>
                  {gradingHistory.length > 0 ? (
                    <div className="history-list">
                      {gradingHistory.map((item) => (
                        <div className="history-row" key={`${item.id}-${item.createdAt}`}>
                          <div>
                            <strong>{decisionLabel(item.action)} · {item.score} 分</strong>
                            <span>{item.actorName || "系统"} · {reviewStageLabel(item.reviewStage)} · {item.modelVersion || "未记录模型"}</span>
                          </div>
                          <p>{item.note || "未填写批注"}</p>
                          <time>{new Date(item.createdAt).toLocaleString("zh-CN", { hour12: false })}</time>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <RequestStateView compact state={{ status: "empty", message: "暂无批阅历史", detail: "保存裁定后会记录教师、时间、分数、阶段和模型版本。" }} />
                  )}
                </section>
              ) : null}
            </>
          ) : (
            <div className="empty-review">
              <ClipboardCheck size={34} />
              <RequestStateView
                compact
                onRetry={() => loadSubjective()}
                state={subjectiveState.status === "empty" ? subjectiveState : { status: "empty", message: "主观题复核已完成", detail: queueStatus }}
              />
            </div>
          )}
        </section>
        ) : null}

        {activeView === "templates" ? (
          <section className="grading-panel">
            <div className="grading-head">
              <div>
                <p className="eyebrow">Template Editor</p>
                <h2>{canvasTitle}</h2>
                <span>{selectedPaperSource?.source ?? "未选择来源"} · {canvasSize.label} · {canvasSize.width} x {canvasSize.height} · 缩放 {Math.round(canvasZoom * 100)}%</span>
              </div>
              {can("template:ai") ? (
                <button className="secondary-button" onClick={() => void generateTemplateAISuggestions()} type="button">
                  <Sparkles size={18} />AI 拆卷建议
                </button>
              ) : null}
            </div>
            <RequestStateView state={templatesState} onRetry={() => loadTemplates()} compact />
            {can("template:ai") && aiSuggestionState.status !== "empty" ? (
              <div className="ai-suggestion-bar">
                <RequestStateView state={aiSuggestionState} compact />
                {aiSuggestedRegions.length > 0 ? (
                  <button className="primary-button" disabled={!canEditSelectedTemplate} onClick={() => void confirmTemplateAISuggestions()} type="button">
                    <Check size={18} />确认写入模板库
                  </button>
                ) : null}
              </div>
            ) : null}
            <div className="template-editor">
              <div className="canvas-shell">
                <div className="canvas-toolbar">
                  <div className="tool-group">
                    {Object.entries(templateTools).map(([key, tool]) => (
                      <button
                        className={activeTool === key ? "tool-button active" : "tool-button"}
                        key={key}
                        onClick={() => setActiveTool(key as TemplateTool)}
                        style={{ "--tool-color": tool.color } as Record<string, string>}
                        type="button"
                      >
                        <span />
                        {tool.label}
                      </button>
                    ))}
                  </div>
                  <div className="tool-group">
                    {canvasPresets.map((preset) => (
                      <button
                        className={canvasSize.label === preset.label ? "template-chip active" : "template-chip"}
                        key={preset.label}
                        onClick={() => {
                          setCanvasSize(preset);
                          setNotice(`已导入${preset.label}模板`);
                        }}
                        type="button"
                      >
                        {preset.label}
                      </button>
                    ))}
                  </div>
                  <div className="zoom-controls">
                    <button className="icon-button" onClick={() => setCanvasZoom((value) => Math.max(0.6, Number((value - 0.1).toFixed(1))))} title="缩小" type="button">-</button>
                    <span>{Math.round(canvasZoom * 100)}%</span>
                    <button className="icon-button" onClick={() => setCanvasZoom((value) => Math.min(1.6, Number((value + 0.1).toFixed(1))))} title="放大" type="button">+</button>
                  </div>
                </div>
                <div className="canvas-viewport">
                  <div
                    className="editable-paper-canvas"
	                    onClick={(event: { clientX: number; clientY: number; target: EventTarget; currentTarget: EventTarget }) => {
	                      if (can("template:edit") && canEditSelectedTemplate) {
	                        void addRegionAt(event);
	                      }
	                    }}
	                    onPointerMove={moveRegionDrag}
	                    onPointerUp={() => void finishRegionDrag()}
                    ref={setCanvasElement}
                    style={{
                      height: `${canvasSize.height * canvasZoom}px`,
                      width: `${canvasSize.width * canvasZoom}px`
                    }}
                  >
                    <div className="blank-paper-scan">
                      <div className="paper-title-line">{canvasTitle}</div>
                    </div>
                    {canvasRegions.map((item) => (
                      <button
                        className={item.id === selectedRegionId ? "canvas-region active" : "canvas-region"}
                        key={item.id}
                        onClick={(event: { stopPropagation: () => void }) => event.stopPropagation()}
                        onPointerDown={(event: {
                          clientX: number;
                          clientY: number;
                          stopPropagation: () => void;
                          currentTarget: { setPointerCapture?: (pointerId: number) => void };
                          pointerId: number;
                        }) => startRegionDrag(event, item, "move")}
                        style={canvasRegionStyle(item)}
                        type="button"
                      >
                        <span>{item.no} · {item.label}</span>
                        <i
                          aria-label="调整大小"
                          onPointerDown={(event: {
                            clientX: number;
                            clientY: number;
                            stopPropagation: () => void;
                            currentTarget: { setPointerCapture?: (pointerId: number) => void };
                            pointerId: number;
                          }) => startRegionDrag(event, item, "resize")}
                        />
                      </button>
                    ))}
                  </div>
                </div>
              </div>
              <div className="template-side">
                <h3>试卷来源</h3>
                <div className="source-toggle">
                  {can("scan:create") ? (
                    <button
                      className={templateSourceMode === "scan" ? "active" : ""}
                      onClick={() => {
                        setTemplateSourceMode("scan");
                        importTemplateScanFromCurrentTask();
                      }}
                      type="button"
                    >
                      导入扫描件
                    </button>
                  ) : null}
                  <button
                    className={templateSourceMode === "library" ? "active" : ""}
                    onClick={() => setTemplateSourceMode("library")}
                    type="button"
                  >
                    从库存选择
                  </button>
                </div>
                {templateSourceMode === "library" ? (
                  <div className="source-list">
                    {paperSources.map((source) => (
                      <button
                        className={source.id === selectedPaperSourceId ? "source-card active" : "source-card"}
                        key={source.id}
                        onClick={() => applyPaperSource(source)}
                        type="button"
                      >
                        <strong>{source.title}</strong>
                        <span>{source.className} · {source.pages} 页 · {source.size.label}</span>
                        <em>{source.importedAt}</em>
                      </button>
                    ))}
                  </div>
                ) : null}
                <h3>模版库</h3>
                <TableToolbar
                  batchLabel="批量发布"
	                  filterOptions={[
	                    { label: "全部模板", value: "all" },
	                    { label: "草稿", value: "status:draft" },
	                    { label: "已发布", value: "status:published" },
	                    { label: "停用", value: "status:disabled" },
	                    { label: "数学", value: "数学" },
	                    { label: "六年级", value: "六年级" }
	                  ]}
                  filterValue={templateFilter}
                  onBatchAction={() => runBatchAction("模板库批量操作", selectedTemplateIds.length)}
                  onFilterChange={(value) => {
                    setTemplateFilter(value);
                    setTemplatePage(1);
                  }}
                  onSearchChange={(value) => {
                    setTemplateSearch(value);
                    setTemplatePage(1);
                  }}
                  onSortChange={(value) => {
                    setTemplateSort(value);
                    setTemplatePage(1);
                  }}
                  searchPlaceholder="模板名称、年级或学科"
                  searchValue={templateSearch}
                  selectedCount={selectedTemplateIds.length}
                  sortOptions={[
                    { label: "名称升序", value: "name_asc" },
                    { label: "总分高到低", value: "score_desc" },
                    { label: "题数多到少", value: "questions_desc" }
                  ]}
                  sortValue={templateSort}
                  totalCount={filteredTemplates.length}
                />
                <div className="template-library-list">
                  {filteredTemplates.length > 0 ? (
                    pagedTemplates.map((template) => (
                      <div className={template.id === selectedTemplateId ? "library-card table-row active" : "library-card table-row"} key={template.id}>
                        <input
                          aria-label={`选择${template.name}`}
                          checked={selectedTemplateIds.includes(template.id)}
                          onChange={() => toggleSelected(template.id, selectedTemplateIds, setSelectedTemplateIds)}
                          type="checkbox"
                        />
	                        <button className="library-card-main" onClick={() => applyTemplateFromLibrary(template)} type="button">
	                          <strong>{template.name}</strong>
	                          <span>{template.grade} · {template.subject} · V{template.version ?? 1} · {template.questionCount} 题 · {template.totalScore} 分</span>
	                          <em className={`status-pill ${normalizeTemplateStatus(template.status)}`}>{templateStatusLabels[normalizeTemplateStatus(template.status)]}</em>
	                        </button>
	                        <div className="library-actions">
	                          <button className="template-chip active" onClick={() => applyTemplateFromLibrary(template)} type="button">引用</button>
	                          <button className="template-chip" onClick={() => copyTemplateFromLibrary(template.id)} type="button">复制新版本</button>
	                          {normalizeTemplateStatus(template.status) === "draft" ? (
	                            <button className="template-chip" onClick={() => updateTemplateStatus(template.id, "published")} type="button">发布</button>
	                          ) : null}
	                          {normalizeTemplateStatus(template.status) === "published" ? (
	                            <button className="template-chip" onClick={() => updateTemplateStatus(template.id, "disabled")} type="button">停用</button>
	                          ) : null}
	                          {normalizeTemplateStatus(template.status) === "disabled" ? (
	                            <button className="template-chip" onClick={() => updateTemplateStatus(template.id, "draft")} type="button">转草稿</button>
	                          ) : null}
	                          {can("template:delete") ? (
	                            <button className="template-chip" onClick={() => deleteTemplateFromLibrary(template.id)} type="button">删除</button>
	                          ) : null}
                        </div>
                      </div>
                    ))
                  ) : (
                    <RequestStateView
                      compact
                      onRetry={() => loadTemplates()}
                      state={{ status: "empty", message: "没有符合条件的模版", detail: "调整搜索或筛选条件后重试。" }}
                    />
                  )}
                </div>
                <TablePagination page={templatePage} total={filteredTemplates.length} onPageChange={setTemplatePage} />
                <h3>模板设置</h3>
                <div className="region-style-editor">
                  <label>
                    模板名称
                    <input
                      onChange={(event: { target: { value: string } }) => setCanvasTitle(event.target.value)}
                      value={canvasTitle}
                    />
                  </label>
                </div>
	                <p>模版库总数：{templates.length} 个</p>
	                <p>当前状态：{templateStatusLabels[selectedTemplateStatus]} · V{selectedTemplate?.version ?? 1}</p>
	                <p>来源文件：{selectedTemplate?.sourceFileUrl || "未绑定"}</p>
	                <p>区域数量：{canvasRegions.length} 个</p>
	                <p>当前题区：{selectedCanvasRegion ? `${selectedCanvasRegion.no} · ${selectedCanvasRegion.label}` : "未选择"}</p>
                <div className="region-style-editor">
                  <label>
                    边框颜色
                    <input
                      disabled={!selectedCanvasRegion}
                      onChange={(event: { target: { value: string } }) => updateSelectedRegion((item) => ({ ...item, color: event.target.value }))}
                      type="color"
                      value={selectedCanvasRegion?.color ?? "#0d7c66"}
                    />
                  </label>
                  <label>
                    线型
                    <select
                      disabled={!selectedCanvasRegion}
                      onChange={(event: { target: { value: string } }) => updateSelectedRegion((item) => ({ ...item, borderStyle: event.target.value as CanvasRegion["borderStyle"] }))}
                      value={selectedCanvasRegion?.borderStyle ?? "solid"}
                    >
                      <option value="solid">实线</option>
                      <option value="dashed">虚线</option>
                      <option value="dotted">点线</option>
	                    </select>
	                  </label>
	                </div>
	                <h3>题目结构</h3>
	                <div className="question-config-editor">
	                  <label>
	                    题号
	                    <input
	                      disabled={!selectedCanvasRegion || !canEditSelectedTemplate}
	                      onChange={(event: { target: { value: string } }) => updateSelectedRegion((item) => ({ ...item, no: event.target.value }))}
	                      value={selectedCanvasRegion?.no ?? ""}
	                    />
	                  </label>
	                  <label>
	                    题型
	                    <select
	                      disabled={!selectedCanvasRegion || !canEditSelectedTemplate}
	                      onChange={(event: { target: { value: string } }) => {
	                        const nextType = event.target.value as TemplateTool;
	                        const nextTool = templateTools[nextType];
	                        updateSelectedRegion((item) => ({
	                          ...item,
	                          type: nextType,
	                          label: nextTool.label,
	                          color: nextTool.color,
	                          score: item.score || (nextType === "subjective" ? 10 : 2)
	                        }));
	                      }}
	                      value={selectedCanvasRegion?.type ?? "subjective"}
	                    >
	                      <option value="choice">选择题</option>
	                      <option value="judge">判断题</option>
	                      <option value="objective">客观题</option>
	                      <option value="subjective">主观题</option>
	                    </select>
	                  </label>
	                  <label>
	                    分值
	                    <input
	                      disabled={!selectedCanvasRegion || !canEditSelectedTemplate}
	                      min="0"
	                      onChange={(event: { target: { value: string } }) => updateSelectedRegion((item) => ({ ...item, score: Number(event.target.value) || 0 }))}
	                      type="number"
	                      value={selectedCanvasRegion?.score ?? 0}
	                    />
	                  </label>
	                  <label>
	                    标准答案
	                    <textarea
	                      disabled={!selectedCanvasRegion || !canEditSelectedTemplate}
	                      onChange={(event: { target: { value: string } }) => updateSelectedRegion((item) => ({ ...item, standardAnswer: event.target.value }))}
	                      rows={3}
	                      value={selectedCanvasRegion?.standardAnswer ?? ""}
	                    />
	                  </label>
	                  {selectedCanvasRegion && selectedCanvasRegion.type !== "subjective" ? (
	                    <div className="objective-config-panel">
	                      <div>
	                        <strong>客观题答案配置</strong>
	                        <span>{selectedCanvasRegion.score} 分 · {selectedCanvasRegion.standardAnswer || "未配置答案"}</span>
	                      </div>
	                      <div className="answer-chip-list">
	                        {objectiveAnswerOptions[selectedCanvasRegion.type].map((answer) => (
	                          <button
	                            className={selectedCanvasRegion.standardAnswer === answer ? "template-chip active" : "template-chip"}
	                            disabled={!canEditSelectedTemplate}
	                            key={`${selectedCanvasRegion.id}-${answer}`}
	                            onClick={() => updateSelectedRegion((item) => ({ ...item, standardAnswer: answer, scoringRules: item.scoringRules.length > 0 ? item.scoringRules : ["答案一致得满分", "缺答或识别异常进入复核"] }))}
	                            type="button"
	                          >
	                            {answer}
	                          </button>
	                        ))}
	                      </div>
	                    </div>
	                  ) : null}
	                  <label>
	                    采分点
	                    <textarea
	                      disabled={!selectedCanvasRegion || !canEditSelectedTemplate}
	                      onChange={(event: { target: { value: string } }) => updateSelectedRegion((item) => ({ ...item, scoringRules: event.target.value.split("\n").map((line) => line.trim()).filter(Boolean) }))}
	                      rows={3}
	                      value={(selectedCanvasRegion?.scoringRules ?? []).join("\n")}
	                    />
	                  </label>
	                  <label>
	                    知识点
	                    <textarea
	                      disabled={!selectedCanvasRegion || !canEditSelectedTemplate}
	                      onChange={(event: { target: { value: string } }) => updateSelectedRegion((item) => ({ ...item, knowledge: event.target.value.split(/[，,\n]/).map((line) => line.trim()).filter(Boolean) }))}
	                      rows={2}
	                      value={(selectedCanvasRegion?.knowledge ?? []).join("，")}
	                    />
	                  </label>
	                  {!canEditSelectedTemplate ? (
	                    <div className="permission-note">当前模板已发布或停用，请复制新版本后编辑题目结构。</div>
	                  ) : null}
	                </div>
	                <h3>草稿箱</h3>
                <div className="draft-list">
                  {templateDrafts.length > 0 ? (
                    templateDrafts.map((draft) => (
                      <button className="source-card" key={draft.id} onClick={() => loadTemplateDraft(draft)} type="button">
                        <strong>{draft.title}</strong>
                        <span>{draft.sourceTitle} · {draft.regions.length} 个区域</span>
                        <em>{draft.updatedAt}</em>
                      </button>
                    ))
                  ) : (
                    <RequestStateView
                      compact
                      state={{ status: "empty", message: "暂无草稿", detail: "保存草稿后可从这里恢复编辑。" }}
                    />
                  )}
                </div>
                <div className="decision-actions">
                  {can("template:edit") ? (
                    <>
	                      <button className="secondary-button" onClick={saveCurrentAsTemplate} type="button"><FileStack size={18} />保存为模版</button>
		                      <button className="secondary-button" disabled={!canEditSelectedTemplate} onClick={updateCurrentTemplate} type="button"><FileStack size={18} />更新模版</button>
		                      <button className="secondary-button" disabled={!canEditSelectedTemplate} onClick={saveCurrentRegions} type="button"><Check size={18} />保存题区</button>
	                      <button className="primary-button" onClick={saveTemplateDraft} type="button"><Check size={18} />保存草稿</button>
		                      <button className="ghost-button" disabled={!canEditSelectedTemplate} onClick={() => void deleteSelectedRegion()} type="button">删除区域</button>
                    </>
                  ) : (
                    <div className="permission-note">当前角色仅可查看模板，不能编辑或保存。</div>
                  )}
                  {can("grading:review") ? (
                    <button className="ghost-button" onClick={openReviewQueue} type="button">进入阅卷</button>
                  ) : null}
                </div>
              </div>
            </div>
          </section>
        ) : null}

        {activeView === "mistakes" ? (
          <section className="insight-grid">
            <div className="panel">
              <div className="panel-head">
                <div>
                  <p className="eyebrow">Wrong Questions</p>
                  <h2>错题归档</h2>
                </div>
                {can("mistake:generate") ? (
                  <button className="primary-button" onClick={() => void createRepracticeTask()} type="button"><BookOpenCheck size={18} />生成再练任务</button>
                ) : null}
              </div>
              <RequestStateView state={mistakesState} onRetry={() => loadWrongQuestions()} compact />
              <TableToolbar
                batchLabel="批量再练"
                filterOptions={[
                  { label: "全部错题", value: "all" },
                  { label: "概念错误", value: "concept" },
                  { label: "计算错误", value: "calculation" },
                  { label: "审题错误", value: "reading" },
                  { label: "表达不完整", value: "expression" }
                ]}
                filterValue={mistakeFilter}
                onBatchAction={() => void createRepracticeTask()}
                onFilterChange={(value) => {
                  setMistakeFilter(value);
                  setMistakePage(1);
                }}
                onSearchChange={(value) => {
                  setMistakeSearch(value);
                  setMistakePage(1);
                }}
                onSortChange={(value) => {
                  setMistakeSort(value);
                  setMistakePage(1);
                }}
                searchPlaceholder="题号、学生、错因"
                searchValue={mistakeSearch}
                selectedCount={selectedMistakeIds.length}
                sortOptions={[
                  { label: "得分率低到高", value: "wrong_desc" },
                  { label: "题号升序", value: "question_asc" },
                  { label: "学生姓名", value: "student_asc" }
                ]}
                sortValue={mistakeSort}
                totalCount={filteredMistakes.length}
              />
              <div className="mistake-filter-grid">
                <label>考试<select value={mistakePaperFilter} onChange={(event: { target: { value: string } }) => setMistakePaperFilter(event.target.value)}><option value="all">全部考试</option>{Array.from(new Set(wrongQuestions.map((item) => item.sourcePaper))).map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
                <label>班级<select value={mistakeClassFilter} onChange={(event: { target: { value: string } }) => setMistakeClassFilter(event.target.value)}><option value="all">全部班级</option>{Array.from(new Set(wrongQuestions.map((item) => item.className))).map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
                <label>学生<select value={mistakeStudentFilter} onChange={(event: { target: { value: string } }) => setMistakeStudentFilter(event.target.value)}><option value="all">全部学生</option>{Array.from(new Set(wrongQuestions.map((item) => item.studentName))).map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
                <label>知识点<select value={mistakeKnowledgeFilter} onChange={(event: { target: { value: string } }) => setMistakeKnowledgeFilter(event.target.value)}><option value="all">全部知识点</option>{Array.from(new Set(wrongQuestions.map((item) => item.knowledgePoint))).map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
              </div>
              {filteredMistakes.length > 0 ? (
                pagedMistakes.map((item) => {
                  const rowId = String(item.id);
                  return (
                  <div className={selectedMistake?.id === item.id ? "mistake-record-row table-row active" : "mistake-record-row table-row"} key={item.id}>
                    <input
                      aria-label={"选择第" + item.questionNo + "题"}
                      checked={selectedMistakeIds.includes(rowId)}
                      onChange={() => toggleSelected(rowId, selectedMistakeIds, setSelectedMistakeIds)}
                      type="checkbox"
                    />
                    <button className="mistake-record-main" onClick={() => void openMistakeDetail(item)} type="button">
                      <strong>第 {item.questionNo} 题 · {item.studentName}</strong>
                      <span>{item.sourcePaper} · {item.knowledgePoint} · {errorTypeLabels[item.errorType] ?? "其他"}</span>
                    </button>
                    <em>{item.score}/{item.maxScore} 分</em>
                    <span className={item.repracticeStatus === "assigned" ? "status-pill published" : "status-pill draft"}>{item.repracticeStatus === "assigned" ? "已布置再练" : "待订正"}</span>
                  </div>
                  );
                })
              ) : (
                <RequestStateView
                  compact
                  onRetry={() => loadWrongQuestions()}
                  state={{ status: "empty", message: "没有符合条件的错题", detail: "调整搜索或筛选条件后重试。" }}
                />
              )}
              <TablePagination page={mistakePage} total={filteredMistakes.length} onPageChange={setMistakePage} />
            </div>
            <div className="panel">
              <div className="panel-head">
                <div>
                  <p className="eyebrow">Knowledge</p>
                  <h2>知识点掌握度</h2>
                </div>
              </div>
              {learningProfile.knowledgeMastery.length > 0 ? (
                learningProfile.knowledgeMastery.map((item) => (
                  <button className="mastery-row row-button" key={item.name} onClick={() => setMistakeKnowledgeFilter(item.name)} type="button">
                    <span>{item.name}</span>
                    <div className="accuracy-bar"><div style={{ width: item.mastery + "%" }} /></div>
                    <strong>{item.mastery}%</strong>
                    <small className={item.trend >= 0 ? "trend-up" : "trend-down"}>{item.trend >= 0 ? "+" : ""}{item.trend}</small>
                    <em>{item.studentCount} 人</em>
                  </button>
                ))
              ) : (
                <RequestStateView
                  compact
                  onRetry={() => loadAnalytics()}
                  state={{ status: "empty", message: "暂无知识点错题", detail: "知识点统计生成后会显示在这里。" }}
                />
              )}
              <h3 className="analytics-section-title">学生预警</h3>
              {learningProfile.studentRisks.map((item) => (
                <button className="warning-row row-button" key={item.studentName + item.risk} onClick={() => void loadGuardianReport(item.studentName)} type="button">
                  <div><strong>{item.studentName}</strong><span>{item.risk}</span></div>
                  <em>{item.weakness.join("、")}</em>
                </button>
              ))}
              <div className="guardian-report">
                <div><span>家长可读报告 · {guardianReport.studentName}</span><strong>{guardianReport.summary}</strong></div>
                <p>建议：{guardianReport.actions.join("；")}</p>
              </div>
            </div>
            {selectedMistake ? (
              <div className="panel mistake-detail-panel">
                <div className="panel-head">
                  <div><p className="eyebrow">Mistake Detail</p><h2>第 {selectedMistake.questionNo} 题完整复盘</h2></div>
                  <button className="secondary-button" onClick={() => void createRepracticeTask()} type="button"><BookOpenCheck size={18} />加入再练</button>
                </div>
                <div className="mistake-detail-meta">
                  <span>{selectedMistake.studentName}</span><span>{selectedMistake.className}</span><span>{selectedMistake.sourcePaper}</span>
                  <span>{selectedMistake.knowledgePoint}</span><span>{errorTypeLabels[selectedMistake.errorType] ?? "其他"}</span>
                  <strong>{selectedMistake.score}/{selectedMistake.maxScore} 分</strong>
                </div>
                <div className="mistake-answer-grid">
                  <div><span>原题</span><p>{selectedMistake.originalQuestion || "原题内容待补充"}</p></div>
                  <div><span>学生答案</span><p>{selectedMistake.studentAnswer || "未作答"}</p></div>
                  <div><span>正确答案</span><p>{selectedMistake.correctAnswer}</p></div>
                  <div><span>错因与解析</span><p>{selectedMistake.wrongReason}。{selectedMistake.explanation}</p></div>
                </div>
              </div>
            ) : null}
          </section>
        ) : null}

        {activeView === "workspace" || activeView === "analytics" ? (
        <section className="insight-grid">
          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Class Analytics</p>
                <h2>{analytics.className} 学情概览</h2>
              </div>
              <div className="analytics-actions">
                <button className="secondary-button" onClick={exportScores} type="button"><FileStack size={18} />导出报表</button>
                <button className="primary-button" onClick={() => void generateScores()} type="button"><Check size={18} />生成成绩</button>
              </div>
            </div>
            <RequestStateView state={analyticsState} onRetry={() => loadAnalytics()} compact />
            <div className="score-summary">
              <div>
                <span>应阅人数</span>
                <strong>{analytics.studentCount}</strong>
              </div>
              <div>
                <span>已生成</span>
                <strong>{analytics.gradedCount}</strong>
              </div>
              <div>
                <span>平均分</span>
                <strong>{analytics.averageScore.toFixed(1)}</strong>
              </div>
              <div>
                <span>最高分</span>
                <strong>{analytics.highestScore.toFixed(0)}</strong>
              </div>
              <div>
                <span>最低分</span>
                <strong>{analytics.lowestScore.toFixed(0)}</strong>
              </div>
              <div>
                <span>完成率</span>
                <strong>{analytics.completionRate}%</strong>
              </div>
              <div>
                <span>及格 / 优秀</span>
                <strong>{analytics.passRate}% / {analytics.excellentRate}%</strong>
              </div>
            </div>
            {analytics.scoreBands.length > 0 ? (
              <div className="score-band-list">
                {analytics.scoreBands.map((item) => {
                  const ratio = analytics.gradedCount > 0 ? Math.round((item.count / analytics.gradedCount) * 100) : 0;
                  return (
                    <div className="score-band-row" key={item.label}>
                      <span>{item.label} 分</span>
                      <div className="accuracy-bar">
                        <div style={{ width: `${ratio}%` }} />
                      </div>
                      <strong>{item.count} 人</strong>
                    </div>
                  );
                })}
              </div>
            ) : null}
            <h3 className="analytics-section-title">题目维度统计</h3>
            <div className="analytics-table-wrap">
              <table className="analytics-table">
                <thead>
                  <tr>
                    <th>题目</th>
                    <th>正确率</th>
                    <th>得分率</th>
                    <th>难度</th>
                    <th>区分度</th>
                    <th>典型错误</th>
                  </tr>
                </thead>
                <tbody>
                  {analytics.questionDetails.length > 0 ? (
                    analytics.questionDetails.slice(0, 8).map((item) => (
                      <tr key={`${item.no}-${item.type}`}>
                        <td>第 {item.no} 题 · {item.type}</td>
                        <td>{item.accuracy}%</td>
                        <td>{item.scoreRate}%</td>
                        <td>{item.difficulty}</td>
                        <td>{item.discrimination}</td>
                        <td>{item.typicalError}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={6}>暂无题目维度统计</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Objective Review</p>
                <h2>客观题异常复核</h2>
              </div>
            </div>
            {analytics.objectiveExceptions.length > 0 ? (
              <div className="exception-list">
                {analytics.objectiveExceptions.slice(0, 5).map((item) => (
                  <div className="exception-row" key={item.id}>
                    <div>
                      <strong>{item.studentName || item.submissionId} · 第 {item.questionNo} 题</strong>
                      <span>{item.reason}</span>
                    </div>
                    <em>{item.answer || "缺答"} · {item.confidence}% · {item.status}</em>
                    <button className="template-chip" onClick={() => confirmObjectiveException(item.id)} type="button">人工确认</button>
                  </div>
                ))}
              </div>
            ) : (
              <RequestStateView
                compact
                onRetry={() => loadAnalytics()}
                state={{ status: "empty", message: "暂无客观题异常", detail: "低置信度、缺答或多选异常会显示在这里。" }}
              />
            )}
            <h3 className="analytics-section-title">学生维度统计</h3>
            <div className="analytics-table-wrap">
              <table className="analytics-table">
                <thead>
                  <tr>
                    <th>排名</th>
                    <th>学生</th>
                    <th>班级</th>
                    <th>总分</th>
                    <th>薄弱知识点</th>
                  </tr>
                </thead>
                <tbody>
                  {analytics.studentScores.length > 0 ? (
                    analytics.studentScores.slice(0, 8).map((item) => (
                      <tr key={`${item.rank}-${item.studentName}`}>
                        <td>{item.rank}</td>
                        <td>{item.studentName}</td>
                        <td>{item.className}</td>
                        <td>{item.score.toFixed(1)}</td>
                        <td>{item.weakness.length > 0 ? item.weakness.join("、") : "暂无"}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={5}>暂无学生成绩画像</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Weak Points</p>
                <h2>薄弱点与学生风险</h2>
              </div>
              {can("guardian:remind") ? (
                <button className="ghost-button" onClick={sendGuardianReminders} type="button"><Send size={16} />提醒家长</button>
              ) : null}
            </div>
            {analytics.knowledgeStats.length > 0 || analytics.studentRisks.length > 0 ? (
              <>
                {analytics.knowledgeStats.slice(0, 3).map((item) => (
                  <div className="knowledge-row" key={item.name}>
                    <span>{item.name}</span>
                    <strong>{item.accuracy}%</strong>
                    <small>{item.wrongCount} 次错误</small>
                  </div>
                ))}
                {analytics.studentRisks.slice(0, 3).map((item) => (
                  <div className="warning-row" key={`${item.studentName}-${item.risk}`}>
                    <div>
                      <strong>{item.studentName}</strong>
                      <span>{item.risk}</span>
                    </div>
                    <em>{item.weakness.join("、")}</em>
                  </div>
                ))}
              </>
            ) : (
              <RequestStateView
                compact
                onRetry={() => loadAnalytics()}
                state={{ status: "empty", message: "暂无薄弱点与风险", detail: "统计生成后会显示知识点和学生风险。" }}
              />
            )}
          </div>
        </section>
        ) : null}
      </main>
    </div>
  );
}

export default App;
