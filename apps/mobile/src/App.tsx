import React, { useMemo, useState } from "react";
import {
  Pressable,
  SafeAreaView,
  ScrollView,
  StatusBar,
  StyleSheet,
  Text,
  View
} from "react-native";

type Tab = "tasks" | "upload" | "mistakes";

const tasks = [
  { id: "t1", title: "六年级数学期中卷", status: "待提交", due: "今晚 20:00", subject: "数学" },
  { id: "t2", title: "分数应用题专项", status: "已批改", due: "昨天", subject: "数学" },
  { id: "t3", title: "几何面积小测", status: "待完成", due: "明天 18:00", subject: "数学" }
];

const mistakes = [
  { id: "m1", no: "15", knowledge: "比例应用题", reason: "比例关系书写不规范", score: "8/10" },
  { id: "m2", no: "18", knowledge: "几何面积", reason: "辅助线理解不足", score: "5/8" },
  { id: "m3", no: "7", knowledge: "分数运算", reason: "通分步骤遗漏", score: "3/5" }
];

function App(): React.JSX.Element {
  const [tab, setTab] = useState<Tab>("tasks");

  const title = useMemo(() => {
    if (tab === "tasks") return "学习任务";
    if (tab === "upload") return "拍照提交";
    return "错题本";
  }, [tab]);

  return (
    <SafeAreaView style={styles.safe}>
      <StatusBar barStyle="dark-content" backgroundColor="#f6f8fb" />
      <View style={styles.header}>
        <View>
          <Text style={styles.eyebrow}>Club 学情</Text>
          <Text style={styles.title}>{title}</Text>
        </View>
        <View style={styles.badge}>
          <Text style={styles.badgeText}>六年级 3 班</Text>
        </View>
      </View>

      <View style={styles.tabs}>
        <TabButton active={tab === "tasks"} label="任务" onPress={() => setTab("tasks")} />
        <TabButton active={tab === "upload"} label="上传" onPress={() => setTab("upload")} />
        <TabButton active={tab === "mistakes"} label="错题" onPress={() => setTab("mistakes")} />
      </View>

      <ScrollView contentContainerStyle={styles.content}>
        {tab === "tasks" && (
          <View style={styles.section}>
            {tasks.map((task) => (
              <View style={styles.row} key={task.id}>
                <View>
                  <Text style={styles.rowTitle}>{task.title}</Text>
                  <Text style={styles.rowMeta}>{task.subject} · 截止 {task.due}</Text>
                </View>
                <Text style={task.status === "待提交" ? styles.statusHot : styles.status}>{task.status}</Text>
              </View>
            ))}
          </View>
        )}

        {tab === "upload" && (
          <View style={styles.uploadBox}>
            <Text style={styles.uploadTitle}>提交本次作业</Text>
            <Text style={styles.uploadCopy}>支持拍照、多图和 PDF。提交后教师可在 Web 端看到扫描识别进度。</Text>
            <Pressable style={styles.primaryAction}>
              <Text style={styles.primaryActionText}>选择照片或拍照</Text>
            </Pressable>
            <View style={styles.uploadSteps}>
              <Text style={styles.stepText}>1. 拍清楚整页边缘</Text>
              <Text style={styles.stepText}>2. 多页按顺序上传</Text>
              <Text style={styles.stepText}>3. 提交后等待批改通知</Text>
            </View>
          </View>
        )}

        {tab === "mistakes" && (
          <View style={styles.section}>
            {mistakes.map((item) => (
              <View style={styles.mistake} key={item.id}>
                <View style={styles.mistakeHead}>
                  <Text style={styles.rowTitle}>第 {item.no} 题</Text>
                  <Text style={styles.statusHot}>{item.score}</Text>
                </View>
                <Text style={styles.rowMeta}>知识点：{item.knowledge}</Text>
                <Text style={styles.reason}>错因：{item.reason}</Text>
                <Pressable style={styles.secondaryAction}>
                  <Text style={styles.secondaryActionText}>再练 5 题</Text>
                </Pressable>
              </View>
            ))}
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

function TabButton(props: { active: boolean; label: string; onPress: () => void }) {
  return (
    <Pressable onPress={props.onPress} style={[styles.tab, props.active && styles.tabActive]}>
      <Text style={[styles.tabText, props.active && styles.tabTextActive]}>{props.label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  safe: {
    backgroundColor: "#f6f8fb",
    flex: 1
  },
  header: {
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between",
    paddingHorizontal: 20,
    paddingTop: 18
  },
  eyebrow: {
    color: "#667085",
    fontSize: 12,
    fontWeight: "700"
  },
  title: {
    color: "#17202a",
    fontSize: 28,
    fontWeight: "800",
    marginTop: 4
  },
  badge: {
    backgroundColor: "#e9f7f3",
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 7
  },
  badgeText: {
    color: "#0d7c66",
    fontSize: 12,
    fontWeight: "700"
  },
  tabs: {
    backgroundColor: "#edf1f6",
    borderRadius: 8,
    flexDirection: "row",
    gap: 4,
    margin: 20,
    padding: 4
  },
  tab: {
    alignItems: "center",
    borderRadius: 6,
    flex: 1,
    paddingVertical: 10
  },
  tabActive: {
    backgroundColor: "#ffffff"
  },
  tabText: {
    color: "#506174",
    fontWeight: "700"
  },
  tabTextActive: {
    color: "#0d7c66"
  },
  content: {
    padding: 20,
    paddingTop: 0
  },
  section: {
    backgroundColor: "#ffffff",
    borderColor: "#e5eaf1",
    borderRadius: 8,
    borderWidth: 1
  },
  row: {
    alignItems: "center",
    borderBottomColor: "#edf1f6",
    borderBottomWidth: 1,
    flexDirection: "row",
    justifyContent: "space-between",
    padding: 16
  },
  rowTitle: {
    color: "#17202a",
    fontSize: 16,
    fontWeight: "800"
  },
  rowMeta: {
    color: "#667085",
    fontSize: 13,
    marginTop: 4
  },
  status: {
    color: "#0d7c66",
    fontSize: 13,
    fontWeight: "800"
  },
  statusHot: {
    color: "#c24133",
    fontSize: 13,
    fontWeight: "800"
  },
  uploadBox: {
    backgroundColor: "#ffffff",
    borderColor: "#e5eaf1",
    borderRadius: 8,
    borderWidth: 1,
    padding: 18
  },
  uploadTitle: {
    color: "#17202a",
    fontSize: 20,
    fontWeight: "800"
  },
  uploadCopy: {
    color: "#667085",
    fontSize: 14,
    lineHeight: 22,
    marginTop: 8
  },
  primaryAction: {
    alignItems: "center",
    backgroundColor: "#0d7c66",
    borderRadius: 8,
    marginTop: 18,
    paddingVertical: 13
  },
  primaryActionText: {
    color: "#ffffff",
    fontWeight: "800"
  },
  uploadSteps: {
    gap: 8,
    marginTop: 18
  },
  stepText: {
    color: "#405164",
    fontSize: 14
  },
  mistake: {
    borderBottomColor: "#edf1f6",
    borderBottomWidth: 1,
    padding: 16
  },
  mistakeHead: {
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between"
  },
  reason: {
    color: "#405164",
    fontSize: 14,
    marginTop: 10
  },
  secondaryAction: {
    alignItems: "center",
    alignSelf: "flex-start",
    backgroundColor: "#eef6ff",
    borderRadius: 8,
    marginTop: 12,
    paddingHorizontal: 14,
    paddingVertical: 9
  },
  secondaryActionText: {
    color: "#155b92",
    fontWeight: "800"
  }
});

export default App;

