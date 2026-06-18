package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type App struct {
	config Config
	store  *Store
}

func NewApp() *App {
	config := LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := OpenStore(ctx, config)
	if err != nil {
		log.Printf("database unavailable, falling back to fixtures: %v", err)
		return &App{config: config}
	}
	return &App{config: config, store: store}
}

func NewRouter() http.Handler {
	app := NewApp()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", app.handleHealth)
	mux.HandleFunc("GET /api/dashboard", app.handleDashboard)
	mux.HandleFunc("GET /api/grading/subjective/current", app.handleCurrentSubjective)
	mux.HandleFunc("GET /api/grading/subjective/reviews/{reviewID}", app.handleReviewSubjective)
	mux.HandleFunc("POST /api/grading/subjective/decision", app.handleSubjectiveDecision)
	mux.HandleFunc("GET /api/templates", app.handleTemplates)
	mux.HandleFunc("GET /api/analytics/classroom", app.handleClassroomAnalytics)
	mux.HandleFunc("GET /api/dev/connections", app.handleDevConnections)
	mux.HandleFunc("POST /api/dev/reset-demo", app.handleResetDemo)

	return withCORS(mux)
}

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"database": app.store != nil,
		"config":   app.config.Public(),
	})
}

func (app *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if app.store != nil {
		data, err := app.store.Dashboard(r.Context())
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("dashboard db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, dashboardFixture())
}

func (app *App) handleCurrentSubjective(w http.ResponseWriter, r *http.Request) {
	if app.store != nil {
		data, err := app.store.CurrentSubjective(r.Context())
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no pending subjective reviews"})
			return
		}
		log.Printf("current subjective db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, subjectiveFixture())
}

func (app *App) handleReviewSubjective(w http.ResponseWriter, r *http.Request) {
	reviewID := r.PathValue("reviewID")
	if reviewID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reviewID is required"})
		return
	}
	if app.store != nil {
		data, err := app.store.SubjectiveByReviewID(r.Context(), reviewID)
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "subjective review not found"})
			return
		}
		log.Printf("subjective review db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, subjectiveFixture())
}

func (app *App) handleSubjectiveDecision(w http.ResponseWriter, r *http.Request) {
	var req GradingDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.SubmissionID == "" || req.QuestionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "submissionId and questionId are required"})
		return
	}
	if app.store != nil {
		data, err := app.store.SaveSubjectiveDecision(r.Context(), req)
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("subjective decision db write failed: %v", err)
	}
	writeJSON(w, http.StatusOK, GradingDecisionResponse{
		Status:       "saved",
		FinalScore:   req.FinalScore,
		NextQuestion: "q_018",
	})
}

func (app *App) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if app.store != nil {
		data, err := app.store.Templates(r.Context())
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("templates db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, templatesFixture())
}

func (app *App) handleClassroomAnalytics(w http.ResponseWriter, r *http.Request) {
	if app.store != nil {
		data, err := app.store.ClassroomAnalytics(r.Context())
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("classroom analytics db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, analyticsFixture())
}

func (app *App) handleDevConnections(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, CheckDevConnections(app.config))
}

func (app *App) handleResetDemo(w http.ResponseWriter, r *http.Request) {
	if app.config.AppEnv == "production" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "reset demo is disabled in production"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusOK, dashboardFixture())
		return
	}
	data, err := app.store.ResetDemo(r.Context())
	if err != nil {
		log.Printf("reset demo failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reset demo failed"})
		return
	}
	writeJSON(w, http.StatusOK, data)
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
