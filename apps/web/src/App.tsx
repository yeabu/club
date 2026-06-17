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

function App() {
  const [dashboard, setDashboard] = useState<DashboardData>(fallbackDashboard);
  const [subjective, setSubjective] = useState<SubjectiveData>(fallbackSubjective);
  const [score, setScore] = useState(fallbackSubjective.ai.score);
  const [note, setNote] = useState("步骤完整，结果正确，表达略不规范。");
  const [savedState, setSavedState] = useState("未保存");
  const [activeMode, setActiveMode] = useState<"review" | "template">("review");

  useEffect(() => {
    fetch("/api/dashboard")
      .then((response) => response.json())
      .then(setDashboard)
      .catch(() => setDashboard(fallbackDashboard));

    fetch("/api/grading/subjective/current")
      .then((response) => response.json())
      .then((data: SubjectiveData) => {
        setSubjective(data);
        setScore(data.ai.score);
      })
      .catch(() => setSubjective(fallbackSubjective));
  }, []);

  const selectedReview = useMemo(() => dashboard.reviewQueue[0], [dashboard.reviewQueue]);

  async function saveDecision(decision: "accepted_ai" | "modified" | "rejected") {
    setSavedState("保存中");
    try {
      await fetch("/api/grading/subjective/decision", {
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
      setSavedState("已保存，下一题已准备");
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
          <a className="active"><LayoutDashboard size={18} />工作台</a>
          <a><ScanLine size={18} />扫描导入</a>
          <a><FileStack size={18} />试卷模板</a>
          <a><ClipboardCheck size={18} />阅卷中心</a>
          <a><BookOpenCheck size={18} />错题集</a>
          <a><UsersRound size={18} />学情分析</a>
        </nav>

        <div className="sidebar-note">
          <Sparkles size={18} />
          <span>AI 只提供建议，教师保留最终评分权。</span>
        </div>
      </aside>

      <main className="main">
        <header className="topbar">
          <div>
            <p className="eyebrow">六年级 3 班 · 今日工作台</p>
            <h1>先处理阅卷，再看学情</h1>
          </div>
          <div className="top-actions">
            <button className="icon-button" title="筛选"><SlidersHorizontal size={18} /></button>
            <button className="icon-button" title="通知"><Bell size={18} /></button>
            <button className="primary-button"><ScanLine size={18} />导入扫描件</button>
          </div>
        </header>

        <section className="metrics-grid">
          {dashboard.metrics.map((metric) => (
            <article className={`metric metric-${metric.tone}`} key={metric.label}>
              <span>{metric.label}</span>
              <strong>{metric.value}</strong>
              <small>{metric.delta}</small>
            </article>
          ))}
        </section>

        <section className="work-grid">
          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Scan Queue</p>
                <h2>扫描处理队列</h2>
              </div>
              <button className="ghost-button">查看全部</button>
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
              <button className="ghost-button"><PenLine size={16} />进入阅卷</button>
            </div>
            <div className="review-list">
              {dashboard.reviewQueue.map((item) => (
                <button className={item.id === selectedReview?.id ? "review-row active" : "review-row"} key={item.id}>
                  <span>{item.studentName}</span>
                  <strong>第 {item.questionNo} 题</strong>
                  <em>{item.aiAdvice}</em>
                  <small>置信度 {item.confidence}%</small>
                </button>
              ))}
            </div>
          </div>
        </section>

        <section className="grading-panel">
          <div className="grading-head">
            <div>
              <p className="eyebrow">Subjective Grading</p>
              <h2>{subjective.paperName} · 第 {subjective.questionNo} 题</h2>
              <span>{subjective.className} · {subjective.studentName}</span>
            </div>
            <div className="segmented">
              <button className={activeMode === "review" ? "active" : ""} onClick={() => setActiveMode("review")}>左右分屏批阅</button>
              <button className={activeMode === "template" ? "active" : ""} onClick={() => setActiveMode("template")}>模板信息</button>
            </div>
          </div>

          {activeMode === "review" ? (
            <div className="split-review">
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
                    onChange={(event) => setScore(Number(event.target.value))}
                    step={0.5}
                    type="number"
                    value={score}
                  />
                </label>
                <label>
                  批注
                  <textarea onChange={(event) => setNote(event.target.value)} value={note} />
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
                <div className="question-region region-choice">1-10 选择题区域</div>
                <div className="question-region region-subjective">15 应用题区域</div>
              </div>
              <div className="template-side">
                <h3>题目区域</h3>
                <p>第 {subjective.questionNo} 题 · 主观题 · 满分 {subjective.fullScore} 分</p>
                <p>知识点：{subjective.standardAnswer.knowledge.join("、")}</p>
                <button className="secondary-button"><FileStack size={18} />编辑模板</button>
              </div>
            </div>
          )}
        </section>

        <section className="insight-grid">
          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Weak Points</p>
                <h2>薄弱知识点</h2>
              </div>
            </div>
            {dashboard.weakPoints.map((item) => (
              <div className="knowledge-row" key={item.name}>
                <span>{item.name}</span>
                <strong>{item.accuracy}%</strong>
                <small>{item.wrongCount} 次错误</small>
              </div>
            ))}
          </div>

          <div className="panel">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Homework Watch</p>
                <h2>作业完成预警</h2>
              </div>
              <button className="ghost-button"><Send size={16} />提醒家长</button>
            </div>
            {dashboard.homeworkWatch.map((item) => (
              <div className="warning-row" key={item.studentName}>
                <div>
                  <strong>{item.studentName}</strong>
                  <span>{item.className} · 连续 {item.missing} 次未提交</span>
                </div>
                <em>{item.guardian}</em>
              </div>
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;

