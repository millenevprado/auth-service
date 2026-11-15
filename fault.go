package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
)

// Fault injection for the incident demo ("A Prova Real").
//
// When FAULT_INJECTION_ENABLED=true, POST /admin/fault (MASTER_KEY protected)
// makes /validate answer HTTP 500. The flag lives only in memory, on purpose:
// it simulates a process stuck in a bad state (e.g. a broken connection pool)
// that a restart fixes. That is exactly what the self-healing runbook does
// (kubectl rollout restart), so the demo shows a real recovery.
//
//	POST   /admin/fault -> start failing
//	DELETE /admin/fault -> stop failing
//	GET    /admin/fault -> current state
var faultActive atomic.Bool

func (a *App) faultHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		faultActive.Store(true)
		log.Printf("FAULT INJECTION ATIVADA: /validate vai responder HTTP 500")
	case http.MethodDelete:
		faultActive.Store(false)
		log.Printf("Fault injection desativada")
	case http.MethodGet:
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"fault_active": faultActive.Load()}); err != nil {
		log.Printf("Erro ao escrever resposta: %v", err)
	}
}

// faultMiddleware fails /validate while the fault is active.
func faultMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if faultActive.Load() && r.URL.Path == "/validate" {
			log.Printf("ERRO: falha injetada em /validate (pool de conexões corrompido simulado)")
			http.Error(w, "Erro interno no auth-service", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}
