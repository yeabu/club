import {
  Bell,
  BookOpenCheck,
  Check,
  ClipboardCheck,
  FileStack,
  LayoutDashboard,
  MessageSquareText,
  PenLine,
  ScanLine,
  Send,
  SlidersHorizontal,
  Sparkles,
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
  pages: number;
  status: string;
  progress: number;
};

type ReviewItem = {
  id: string;
  studentName: string;
  paperName: string;
  questionNo: string;
  aiAdvice: string;
  confidence: number;
};

type KnowledgeStat = {
  name: string;
  accuracy: number;
  wrongCount: number;
};

type HomeworkWatch = {
  studentName: string;
  className: string;
  missing: number;
  guardian: string;
};

type DashboardData = {
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
  };
};

type GradingDecisionResponse = {
  status: string;
  finalScore: number;
  nextQuestion: string;
  nextReview?: SubjectiveData;
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
  questions: QuestionTemplate[];
};

type QuestionStat = {
  no: string;
  accuracy: number;
  type: string;
};

type StudentRisk = {
  studentName: string;
  risk: string;
  weakness: string[];
};

type ClassroomAnalytics = {
  className: string;
  averageScore: number;
  highestScore: number;
  lowestScore: number;
  questionStats: QuestionStat[];
  knowledgeStats: KnowledgeStat[];
  studentRisks: StudentRisk[];
};

type ActiveView = "workspace" | "scan" | "templates" | "grading" | "mistakes" | "analytics";
type Overlay = "filter" | "notifications" | null;
type TemplateTool = "objective" | "subjective" | "choice" | "judge";

type CanvasRegion = {
  id: string;
  no: string;
  type: TemplateTool;
  label: string;
  color: string;
  borderStyle: "solid" | "dashed" | "dotted";
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

const canvasPresets: CanvasSize[] = [
  { label: "A4 空白卷", width: 760, height: 1080 },
  { label: "答题卡", width: 760, height: 900 },
  { label: "横向试卷", width: 1080, height: 760 }
];

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

const fallbackDashboard: DashboardData = {
  metrics: [
    { label: "待批试卷", value: "128", delta: "较昨日 +24", tone: "primary" },
    { label: "主观题待复核", value: "36", delta: "AI 已预评分", tone: "warning" },
    { label: "未提交作业", value: "8", delta: "3 人连续未交", tone: "danger" },
    { label: "班级平均分", value: "81.6", delta: "较上次 +3.2", tone: "success" }
  ],
  scanQueue: [
    { id: "scan_001", title: "六年级数学期中卷", className: "六年级 3 班", pages: 96, status: "OCR 识别中", progress: 68 },
    { id: "scan_002", title: "分数应用题专项", className: "六年级 1 班", pages: 42, status: "等待 OMR", progress: 32 },
    { id: "scan_003", title: "几何面积小测", className: "五年级 2 班", pages: 48, status: "待导入", progress: 0 }
  ],
  reviewQueue: [
    { id: "review_001", studentName: "张三", paperName: "六年级数学期中卷", questionNo: "15", aiAdvice: "8 / 10", confidence: 86 },
    { id: "review_002", studentName: "李四", paperName: "六年级数学期中卷", questionNo: "18", aiAdvice: "6 / 8", confidence: 78 },
    { id: "review_003", studentName: "王五", paperName: "分数应用题专项", questionNo: "7", aiAdvice: "4 / 6", confidence: 74 }
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
    questions: [
      { id: "q_001", no: "1", type: "single_choice", score: 2, knowledge: ["分数"], region: { page: 1, x: 120, y: 260, width: 480, height: 80 } },
      { id: "q_015", no: "15", type: "subjective", score: 10, knowledge: ["比例", "应用题建模"], region: { page: 2, x: 96, y: 420, width: 620, height: 180 } }
    ]
  }
];

const fallbackAnalytics: ClassroomAnalytics = {
  className: "六年级 3 班",
  averageScore: 81.6,
  highestScore: 98,
  lowestScore: 54,
  questionStats: [
    { no: "1", accuracy: 96, type: "单选题" },
    { no: "8", accuracy: 82, type: "填空题" },
    { no: "15", accuracy: 42, type: "应用题" },
    { no: "18", accuracy: 38, type: "应用题" }
  ],
  knowledgeStats: fallbackDashboard.weakPoints,
  studentRisks: [
    { studentName: "李四", risk: "连续 3 次未提交作业", weakness: ["分数应用题", "比例"] },
    { studentName: "赵六", risk: "本次低于班均 18 分", weakness: ["几何面积"] }
  ]
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
      region: question.region
    };
  });
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

function App() {
  const [dashboard, setDashboard] = useState<DashboardData>(fallbackDashboard);
  const [subjective, setSubjective] = useState<SubjectiveData | null>(fallbackSubjective);
  const [templates, setTemplates] = useState<PaperTemplate[]>(loadStoredTemplateLibrary);
  const [analytics, setAnalytics] = useState<ClassroomAnalytics>(fallbackAnalytics);
  const [selectedTemplateId, setSelectedTemplateId] = useState(fallbackTemplates[0].id);
  const [selectedReviewId, setSelectedReviewId] = useState(fallbackSubjective.reviewId);
  const [activeView, setActiveView] = useState<ActiveView>("workspace");
  const [overlay, setOverlay] = useState<Overlay>(null);
  const [notice, setNotice] = useState("已连接本地开发环境");
  const [scanTitle, setScanTitle] = useState("六年级数学期中卷");
  const [scanClassName, setScanClassName] = useState("六年级 3 班");
  const [scanPages, setScanPages] = useState(48);
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

  useEffect(() => {
    loadDashboard();
    loadSubjective();
    loadTemplates();
    loadAnalytics();
  }, []);

  const selectedTemplate = useMemo(
    () => templates.find((item) => item.id === selectedTemplateId) ?? templates[0],
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

  const selectedPaperSource = useMemo(
    () => paperSources.find((item) => item.id === selectedPaperSourceId),
    [paperSources, selectedPaperSourceId]
  );

  const viewCopy = {
    workspace: { eyebrow: "六年级 3 班 · 今日工作台", title: "先处理阅卷，再看学情" },
    scan: { eyebrow: "Scan Import", title: "导入扫描件并进入 OCR 队列" },
    templates: { eyebrow: "Paper Templates", title: "试卷模版" },
    grading: { eyebrow: "Grading Center", title: "主观题左右分屏批阅" },
    mistakes: { eyebrow: "Wrong Questions", title: "沉淀错题和薄弱知识点" },
    analytics: { eyebrow: "Class Analytics", title: "查看班级学情画像" }
  } satisfies Record<ActiveView, { eyebrow: string; title: string }>;

  function openView(view: ActiveView) {
    setActiveView(view);
    setOverlay(null);
    if (view === "grading") {
      setActiveMode("review");
    }
    if (view === "templates") {
      setActiveMode("template");
    }
  }

  function toggleOverlay(next: Exclude<Overlay, null>) {
    setOverlay((current) => current === next ? null : next);
  }

  function submitScanImport() {
    const nextJob: ScanJob = {
      id: `scan_local_${Date.now()}`,
      title: scanTitle || "未命名扫描任务",
      className: scanClassName || "未选择班级",
      pages: Number(scanPages) || 1,
      status: "待 OCR",
      progress: 0
    };
    setDashboard((current) => ({
      ...current,
      scanQueue: [nextJob, ...current.scanQueue]
    }));
    const nextSource: TemplatePaperSource = {
      id: `paper_local_${Date.now()}`,
      title: nextJob.title,
      className: nextJob.className,
      pages: nextJob.pages,
      size: canvasPresets[1],
      importedAt: "刚刚",
      source: "现场扫描"
    };
    setPaperSources((current) => [nextSource, ...current]);
    setNotice(`${nextJob.title} 已加入扫描队列`);
    setActiveView("workspace");
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

  function canvasRegionsToQuestions(regions: CanvasRegion[], templateID: string): QuestionTemplate[] {
    return regions.map((item, index) => ({
      id: `${templateID}_${item.id}`,
      no: item.no || `${index + 1}`,
      type: item.type,
      score: item.type === "subjective" ? 10 : 2,
      knowledge: [],
      region: item.region
    }));
  }

  function saveCurrentAsTemplate() {
    const title = canvasTitle.trim() || "未命名试卷模版";
    const templateID = `tpl_local_${Date.now()}`;
    const questions = canvasRegionsToQuestions(canvasRegions, templateID);
    const nextTemplate: PaperTemplate = {
      id: templateID,
      name: title,
      subject: "数学",
      grade: "六年级",
      questionCount: Math.max(questions.length, 1),
      totalScore: questions.reduce((sum, item) => sum + item.score, 0),
      questions
    };
    persistTemplateLibrary([nextTemplate, ...templates].slice(0, 12));
    setSelectedTemplateId(templateID);
    setNotice(`${title} 已保存到模版库`);
  }

  function applyTemplateFromLibrary(template: PaperTemplate) {
    const regions = regionsFromTemplate(template);
    setSelectedTemplateId(template.id);
    setCanvasTitle(template.name);
    setCanvasRegions(regions);
    setSelectedRegionId(regions[0]?.id ?? "");
    setNotice(`已引用模版：${template.name}`);
  }

  function deleteTemplateFromLibrary(templateID: string) {
    const nextTemplates = templates.filter((item) => item.id !== templateID);
    persistTemplateLibrary(nextTemplates);
    setSelectedTemplateId((current) => current === templateID ? (nextTemplates[0]?.id ?? "") : current);
    setNotice("已从模版库删除");
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

  function sendGuardianReminders() {
    const names = analytics.studentRisks.map((item) => item.studentName).join("、");
    setNotice(names ? `已生成 ${names} 的家长提醒` : "当前没有需要提醒的学生");
  }

  async function loadDashboard() {
    try {
      const response = await fetch("/api/dashboard");
      if (!response.ok) {
        throw new Error("dashboard api failed");
      }
      const data = await response.json() as DashboardData;
      setDashboard(data);
      return data;
    } catch {
      setDashboard(fallbackDashboard);
      return fallbackDashboard;
    }
  }

  async function loadTemplates() {
    try {
      const response = await fetch("/api/templates");
      if (!response.ok) {
        throw new Error("templates api failed");
      }
      const data = await response.json() as PaperTemplate[];
      if (templates.length === 0 && data.length > 0) {
        persistTemplateLibrary(data);
      }
      if (data[0] && !selectedTemplateId) {
        setSelectedTemplateId(data[0].id);
      }
      return data;
    } catch {
      setTemplates(fallbackTemplates);
      return fallbackTemplates;
    }
  }

  async function loadAnalytics() {
    try {
      const response = await fetch("/api/analytics/classroom");
      if (!response.ok) {
        throw new Error("analytics api failed");
      }
      const data = await response.json() as ClassroomAnalytics;
      setAnalytics(data);
      return data;
    } catch {
      setAnalytics(fallbackAnalytics);
      return fallbackAnalytics;
    }
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

  function addRegionAt(event: { clientX: number; clientY: number; target: EventTarget; currentTarget: EventTarget }) {
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
  }

  function updateSelectedRegion(updater: (region: CanvasRegion) => CanvasRegion) {
    setCanvasRegions((current) => current.map((item) => item.id === selectedRegionId ? updater(item) : item));
  }

  function startRegionDrag(
    event: { clientX: number; clientY: number; stopPropagation: () => void; currentTarget: { setPointerCapture?: (pointerId: number) => void }; pointerId: number },
    item: CanvasRegion,
    mode: "move" | "resize"
  ) {
    event.stopPropagation();
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

  function deleteSelectedRegion() {
    if (!selectedRegionId) {
      return;
    }
    setCanvasRegions((current) => current.filter((item) => item.id !== selectedRegionId));
    setSelectedRegionId("");
    setNotice("已删除选中区域");
  }

  function applySubjective(data: SubjectiveData) {
    setSubjective(data);
    setSelectedReviewId(data.reviewId);
    setScore(data.ai.score);
    setNote(data.ai.reason);
    setQueueStatus("已连接数据库队列");
    setSavedState("未保存");
  }

  async function loadSubjective(reviewId?: string) {
    const endpoint = reviewId
      ? `/api/grading/subjective/reviews/${encodeURIComponent(reviewId)}`
      : "/api/grading/subjective/current";
    setIsReviewLoading(true);
    try {
      const response = await fetch(endpoint);
      if (response.status === 404) {
        setSubjective(null);
        setSelectedReviewId("");
        setQueueStatus("当前没有待复核主观题");
        setSavedState("队列已清空");
        return null;
      }
      if (!response.ok) {
        throw new Error("subjective api failed");
      }
      const data = await response.json() as SubjectiveData;
      applySubjective(data);
      return data;
    } catch {
      applySubjective(fallbackSubjective);
      setQueueStatus("API 未连接，显示本地示例");
      return fallbackSubjective;
    } finally {
      setIsReviewLoading(false);
    }
  }

  async function openReview(item: ReviewItem) {
    setSavedState("加载中");
    await loadSubjective(item.id);
  }

  async function saveDecision(decision: "accepted_ai" | "modified" | "rejected") {
    if (!subjective) {
      setSavedState("没有可保存的复核项");
      return;
    }
    setSavedState("保存中");
    try {
      const response = await fetch("/api/grading/subjective/decision", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          submissionId: subjective.submissionId,
          questionId: subjective.questionId,
          finalScore: score,
          decision,
          teacherNote: note
        })
      });
      if (!response.ok) {
        throw new Error("decision api failed");
      }
      const result = await response.json() as GradingDecisionResponse;
      const nextDashboard = await loadDashboard();
      if (result.nextReview) {
        applySubjective(result.nextReview);
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
    } catch {
      setSavedState("本地已记录，API 未连接");
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
          <button className={activeView === "workspace" ? "active" : ""} onClick={() => openView("workspace")} type="button">
            <LayoutDashboard size={18} />工作台
          </button>
          <button className={activeView === "scan" ? "active" : ""} onClick={() => openView("scan")} type="button">
            <ScanLine size={18} />扫描导入
          </button>
          <button className={activeView === "templates" ? "active" : ""} onClick={() => openView("templates")} type="button">
            <FileStack size={18} />试卷模板
          </button>
          <button className={activeView === "grading" ? "active" : ""} onClick={() => openView("grading")} type="button">
            <ClipboardCheck size={18} />阅卷中心
          </button>
          <button className={activeView === "mistakes" ? "active" : ""} onClick={() => openView("mistakes")} type="button">
            <BookOpenCheck size={18} />错题集
          </button>
          <button className={activeView === "analytics" ? "active" : ""} onClick={() => openView("analytics")} type="button">
            <UsersRound size={18} />学情分析
          </button>
        </nav>

        <div className="sidebar-note">
          <Sparkles size={18} />
          <span>AI 只提供建议，教师保留最终评分权。</span>
        </div>
      </aside>

      <main className="main">
        <header className="topbar">
          <div>
            <p className="eyebrow">{viewCopy[activeView].eyebrow}</p>
            <h1>{viewCopy[activeView].title}</h1>
            {activeView !== "templates" ? <span className="top-notice">{notice}</span> : null}
          </div>
          <div className="top-actions">
            <button className="icon-button" onClick={() => toggleOverlay("filter")} title="筛选" type="button"><SlidersHorizontal size={18} /></button>
            <button className="icon-button" onClick={() => toggleOverlay("notifications")} title="通知" type="button"><Bell size={18} /></button>
            <button
              className="primary-button"
              onClick={activeView === "templates" ? importTemplateScanFromCurrentTask : () => openView("scan")}
              type="button"
            >
              <ScanLine size={18} />导入扫描件
            </button>
          </div>
          {overlay ? (
            <div className="floating-panel">
              {overlay === "filter" ? (
                <>
                  <p className="eyebrow">Filters</p>
                  <h3>工作台筛选</h3>
                  <button className="template-chip active" onClick={() => setNotice("已筛选六年级 3 班")} type="button">六年级 3 班</button>
                  <button className="template-chip" onClick={() => setNotice("已筛选今日任务")} type="button">今日任务</button>
                  <button className="template-chip" onClick={() => setNotice("已筛选主观题优先")} type="button">主观题优先</button>
                </>
              ) : (
                <>
                  <p className="eyebrow">Notifications</p>
                  <h3>待处理提醒</h3>
                  <div className="notice-row">主观题待复核：{dashboard.reviewQueue.length} 条</div>
                  <div className="notice-row">扫描队列：{dashboard.scanQueue.length} 个任务</div>
                  <div className="notice-row">学生风险：{analytics.studentRisks.length} 条</div>
                </>
              )}
            </div>
          ) : null}
        </header>

        {activeView !== "templates" ? (
          <section className="metrics-grid">
            {dashboard.metrics.map((metric) => (
              <article className={`metric metric-${metric.tone}`} key={metric.label}>
                <span>{metric.label}</span>
                <strong>{metric.value}</strong>
                <small>{metric.delta}</small>
              </article>
            ))}
          </section>
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
                试卷名称
                <input onChange={(event: { target: { value: string } }) => setScanTitle(event.target.value)} value={scanTitle} />
              </label>
              <label>
                班级
                <input onChange={(event: { target: { value: string } }) => setScanClassName(event.target.value)} value={scanClassName} />
              </label>
              <label>
                页数
                <input min={1} onChange={(event: { target: { value: string } }) => setScanPages(Number(event.target.value))} type="number" value={scanPages} />
              </label>
            </div>
            <div className="upload-zone">
              <ScanLine size={28} />
              <strong>扫描件暂存区</strong>
              <span>当前先模拟导入任务，后续接 OBS/MinIO 上传和 OCR Worker。</span>
              <div className="top-actions">
                <button className="secondary-button" onClick={() => setNotice("已选择本地扫描件样例")} type="button">选择文件</button>
                <button className="primary-button" onClick={submitScanImport} type="button">开始导入</button>
              </div>
            </div>
            <div className="scan-list">
              {dashboard.scanQueue.map((job) => (
                <div className="scan-row" key={job.id}>
                  <div>
                    <strong>{job.title}</strong>
                    <span>{job.className} · {job.pages} 页 · {job.status}</span>
                  </div>
                  <div className="progress-wrap" aria-label={`${job.progress}%`}>
                    <div className="progress-track">
                      <div className="progress-fill" style={{ width: `${job.progress}%` }} />
                    </div>
                    <em>{job.progress}%</em>
                  </div>
                </div>
              ))}
            </div>
          </section>
        ) : null}

        {activeView === "workspace" ? (
        <section className="work-grid">
          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Scan Queue</p>
                <h2>扫描处理队列</h2>
              </div>
              <button className="ghost-button" onClick={() => openView("scan")} type="button">查看全部</button>
            </div>
            <div className="scan-list">
              {dashboard.scanQueue.map((job) => (
                <div className="scan-row" key={job.id}>
                  <div>
                    <strong>{job.title}</strong>
                    <span>{job.className} · {job.pages} 页 · {job.status}</span>
                  </div>
                  <div className="progress-wrap" aria-label={`${job.progress}%`}>
                    <div className="progress-track">
                      <div className="progress-fill" style={{ width: `${job.progress}%` }} />
                    </div>
                    <em>{job.progress}%</em>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Review Queue</p>
                <h2>主观题复核</h2>
              </div>
              <button className="ghost-button" onClick={() => openView("grading")} type="button"><PenLine size={16} />进入阅卷</button>
            </div>
            <div className="review-list">
              {dashboard.reviewQueue.length > 0 ? (
                dashboard.reviewQueue.map((item) => (
                  <button
                    className={item.id === selectedReview?.id ? "review-row active" : "review-row"}
                    key={item.id}
                    onClick={() => openReview(item)}
                    type="button"
                  >
                    <span>{item.studentName}</span>
                    <strong>第 {item.questionNo} 题</strong>
                    <em>{item.aiAdvice}</em>
                    <small>置信度 {item.confidence}%</small>
                  </button>
                ))
              ) : (
                <div className="empty-state">暂无待复核主观题</div>
              )}
            </div>
          </div>
        </section>
        ) : null}

        {activeView === "workspace" || activeView === "grading" ? (
        <section className="grading-panel">
          {subjective ? (
            <>
              <div className="grading-head">
                <div>
                  <p className="eyebrow">Subjective Grading</p>
                  <h2>{subjective.paperName} · 第 {subjective.questionNo} 题</h2>
                  <span>{subjective.className} · {subjective.studentName} · {queueStatus}</span>
                </div>
                <div className="segmented">
                  <button className={activeMode === "review" ? "active" : ""} onClick={() => setActiveMode("review")}>左右分屏批阅</button>
                  <button className={activeMode === "template" ? "active" : ""} onClick={() => setActiveMode("template")}>模板信息</button>
                </div>
              </div>

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
                        <div className="rule-row" key={rule}>{rule}</div>
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
                    <div className="student-paper">
                      <div className="paper-line wide" />
                      <div className="paper-line medium" />
                      <p>{subjective.studentAnswer.ocrText}</p>
                      <div className="paper-line short" />
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
                    <label>
                      批注
                      <textarea onChange={(event: { target: { value: string } }) => setNote(event.target.value)} value={note} />
                    </label>
                    <div className="decision-actions">
                      <button className="primary-button" onClick={() => saveDecision("accepted_ai")}><Check size={18} />接受 AI</button>
                      <button className="secondary-button" onClick={() => saveDecision("modified")}><PenLine size={18} />修改保存</button>
                      <button className="ghost-button" onClick={() => saveDecision("rejected")}>驳回建议</button>
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
                    <button className="secondary-button" onClick={() => openView("templates")} type="button"><FileStack size={18} />编辑模板</button>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="empty-review">
              <ClipboardCheck size={34} />
              <h2>主观题复核已完成</h2>
              <p>{queueStatus}</p>
              <button className="secondary-button" onClick={() => loadDashboard()}>刷新工作台</button>
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
              <button className="secondary-button" onClick={() => setNotice("AI 拆卷建议已生成，等待教师确认")} type="button">
                <Sparkles size={18} />AI 拆卷建议
              </button>
            </div>
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
                    onClick={addRegionAt}
                    onPointerMove={moveRegionDrag}
                    onPointerUp={() => setDragState(null)}
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
                <div className="template-library-list">
                  {templates.length > 0 ? (
                    templates.map((template) => (
                      <div className={template.id === selectedTemplateId ? "library-card active" : "library-card"} key={template.id}>
                        <button className="library-card-main" onClick={() => applyTemplateFromLibrary(template)} type="button">
                          <strong>{template.name}</strong>
                          <span>{template.grade} · {template.subject} · {template.questionCount} 题 · {template.totalScore} 分</span>
                        </button>
                        <div className="library-actions">
                          <button className="template-chip active" onClick={() => applyTemplateFromLibrary(template)} type="button">引用</button>
                          <button className="template-chip" onClick={() => deleteTemplateFromLibrary(template.id)} type="button">删除</button>
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className="empty-state">暂无可用模版</div>
                  )}
                </div>
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
                    <div className="empty-state">暂无草稿</div>
                  )}
                </div>
                <div className="decision-actions">
                  <button className="secondary-button" onClick={saveCurrentAsTemplate} type="button"><FileStack size={18} />保存为模版</button>
                  <button className="primary-button" onClick={saveTemplateDraft} type="button"><Check size={18} />保存草稿</button>
                  <button className="ghost-button" onClick={deleteSelectedRegion} type="button">删除区域</button>
                  <button className="ghost-button" onClick={() => openView("grading")} type="button">进入阅卷</button>
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
                <button className="primary-button" onClick={() => setNotice("已生成本班错题本")} type="button"><BookOpenCheck size={18} />生成错题本</button>
              </div>
              {analytics.questionStats.map((item) => (
                <div className="question-stat-row mistake-stat-row" key={`mistake-${item.no}-${item.type}`}>
                  <span>第 {item.no} 题 · {item.type}</span>
                  <div className="accuracy-bar">
                    <div style={{ width: `${item.accuracy}%` }} />
                  </div>
                  <strong>{100 - item.accuracy}% 错误</strong>
                </div>
              ))}
            </div>
            <div className="panel">
              <div className="panel-head">
                <div>
                  <p className="eyebrow">Knowledge</p>
                  <h2>按知识点整理</h2>
                </div>
              </div>
              {analytics.knowledgeStats.map((item) => (
                <button className="knowledge-row row-button" key={`wrong-${item.name}`} onClick={() => setNotice(`已筛选 ${item.name} 错题`)} type="button">
                  <span>{item.name}</span>
                  <strong>{item.accuracy}%</strong>
                  <small>{item.wrongCount} 次错误</small>
                </button>
              ))}
            </div>
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
            </div>
            <div className="score-summary">
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
            </div>
            <div className="question-stat-list">
              {analytics.questionStats.slice(0, 5).map((item) => (
                <div className="question-stat-row" key={`${item.no}-${item.type}`}>
                  <span>第 {item.no} 题 · {item.type}</span>
                  <div className="accuracy-bar">
                    <div style={{ width: `${item.accuracy}%` }} />
                  </div>
                  <strong>{item.accuracy}%</strong>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Weak Points</p>
                <h2>薄弱点与学生风险</h2>
              </div>
              <button className="ghost-button" onClick={sendGuardianReminders} type="button"><Send size={16} />提醒家长</button>
            </div>
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
          </div>
        </section>
        ) : null}
      </main>
    </div>
  );
}

export default App;
