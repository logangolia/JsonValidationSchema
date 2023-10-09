package database

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type subHandler struct {
	mu sync.Mutex
}

type writeFlusher interface {
	http.ResponseWriter
	http.Flusher
}

func NewSubHandler() *subHandler {
	return &subHandler{}
}

func (s *subHandler) send(wf writeFlusher, event string, data string) {
	// Create event
	var evt bytes.Buffer

	if event == "comment" {
		evt.WriteString("")
	} else {
		s.mu.Lock()
		defer s.mu.Unlock()
		id := fmt.Sprint(time.Now().UnixMilli())
		evt.WriteString(fmt.Sprintf("event: " + event + "\ndata: " + data + "\nid: " + id))
	}

	slog.Info("Sending", "msg", evt.String())

	// Send event
	wf.Write(evt.Bytes())
	wf.Flush()
}

func (s *subHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wf, ok := w.(writeFlusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	slog.Info("Converted to writeFlusher")

	// Set up event stream connection
	wf.Header().Set("Content-Type", "text/event-stream")
	wf.Header().Set("Cache-Control", "no-cache")
	wf.Header().Set("Connection", "keep-alive")
	wf.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
	wf.Header().Set("Access-Control-Allow-Origin", "*")
	wf.Header().Set("Authorization", "Bearer")
	wf.WriteHeader(http.StatusOK)
	wf.Flush()

	slog.Info("Event stream successfully setup")

	for {
		select {
		case <-r.Context().Done():
			// Client closed connection
			slog.Info("Client closed connection")
			return

		case <-time.After(15 * time.Second):
			// Send next count
			s.send(wf, "comment", "")
		}
	}
}

func (s *subHandler) MessageHandler(w http.ResponseWriter, r *http.Request) {
	// Convert ResponseWriter to a writeFlusher
	wf, ok := w.(writeFlusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		//Delete has been called in or on something in the path
		slog.Info("Delete called on" + r.URL.Path)

		data := r.URL.Path

		s.send(wf, "delete", data)
	case http.MethodPost:
		//Collection only, someone posted to this collection
		slog.Info("Post called on" + r.URL.Path)
		s.send(wf, "update", r.URL.Path)
	case http.MethodPatch:
		//Document is updated
		slog.Info("Patch called on" + r.URL.Path)
		s.send(wf, "update", r.URL.Path)
	case http.MethodPut:
		//Document is created or replaced at Path
		slog.Info("Put called on" + r.URL.Path)
		s.send(wf, "update", r.URL.Path)
	}
}
