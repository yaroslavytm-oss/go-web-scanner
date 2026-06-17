package server

import (
	"context"
	"encoding/json"
	"net/http"
	"scanner/pkg/engine"
	"sync"
)

type WebServer struct {
	scanner    *engine.Scanner
	listenAddr string
	scanCancel context.CancelFunc
	mu         sync.Mutex
}

func NewWebServer(addr string, scanner *engine.Scanner) *WebServer {
	return &WebServer{
		listenAddr: addr,
		scanner:    scanner,
	}
}

type ScanRequest struct {
	Path string `json:"path"`
}

// Start ініціалізує роути та запускає HTTP сервер
func (ws *WebServer) Start(fsRoot http.FileSystem) error {
	mux := http.NewServeMux()

	// Статичний фронтенд вшитий через embed
	mux.Handle("/", http.FileServer(fsRoot))

	// API ендпоінти
	mux.HandleFunc("/api/status", ws.handleStatus)
	mux.HandleFunc("/api/scan", ws.handleScan)
	mux.HandleFunc("/api/results", ws.handleResults)

	return http.ListenAndServe(ws.listenAddr, mux)
}

func (ws *WebServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	isScanning, scanned, found, _ := ws.scanner.GetStats()
	statusStr := "Idle"
	if isScanning {
		statusStr = "Scanning"
	}

	resp := map[string]interface{}{
		"status":        statusStr,
		"files_scanned": scanned,
		"viruses_found": found,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebServer) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Path == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ws.mu.Lock()
	defer ws.mu.Unlock()

	isScanning, _, _, _ := ws.scanner.GetStats()
	if isScanning {
		http.Error(w, "Scan already in progress", http.StatusConflict)
		return
	}

	// Ініціалізація контексту з можливістю скасування для нової таски
	ctx, cancel := context.WithCancel(context.Background())
	ws.scanCancel = cancel

	go func() {
		ws.scanner.StartScan(ctx, req.Path)
	}()

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"message": "Scan started successfully"}`))
}

func (ws *WebServer) handleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _, _, infected := ws.scanner.GetStats()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(infected)
}
