package app

import (
	"encoding/json"
	"net/http"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/dashboard", handleDashboard)
	mux.HandleFunc("GET /api/grading/subjective/current", handleCurrentSubjective)
	mux.HandleFunc("POST /api/grading/subjective/decision", handleSubjectiveDecision)
	mux.HandleFunc("GET /api/templates", handleTemplates)
	mux.HandleFunc("GET /api/analytics/classroom", handleClassroomAnalytics)

	return withCORS(mux)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, dashboardFixture())
}

func handleCurrentSubjective(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, subjectiveFixture())
}

func handleSubjectiveDecision(w http.ResponseWriter, r *http.Request) {
	var req GradingDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.SubmissionID == "" || req.QuestionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "submissionId and questionId are required"})
		return
	}
	writeJSON(w, http.StatusOK, GradingDecisionResponse{
		Status:       "saved",
		FinalScore:   req.FinalScore,
		NextQuestion: "q_018",
	})
}

func handleTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, templatesFixture())
}

func handleClassroomAnalytics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, analyticsFixture())
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
