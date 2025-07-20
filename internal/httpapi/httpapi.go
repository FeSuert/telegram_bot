package httpapi

import (
	"encoding/json"
	"net/http"

	"home-alarm-bot/internal/state"
	"home-alarm-bot/internal/telegram"
)

type Server struct {
	store *state.Store
	bot   *telegram.Bot
}

func New(store *state.Store, bot *telegram.Bot) *Server {
	return &Server{store: store, bot: bot}
}

func (s *Server) Listen(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/arm", func(w http.ResponseWriter, r *http.Request) {
		s.store.Set(state.Armed)
		s.bot.Broadcast("🔒 System Armed (via local API)")
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/disarm", func(w http.ResponseWriter, r *http.Request) {
		s.store.Set(state.Disarmed)
		s.bot.Broadcast("🔓 System Disarmed (via local API)")
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			State state.AlarmState `json:"state"`
		}{s.store.Get()})
	})

	mux.HandleFunc("/alarm", func(w http.ResponseWriter, r *http.Request) {
		s.bot.Broadcast("🚨 **ALARM TRIGGERED**")
	})

	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		s.bot.Broadcast("**System disarmed via PIN**")
	})

	mux.HandleFunc("/video", s.handleVideo)
	

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleVideo(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "invalid multipart form", http.StatusBadRequest)
        return
    }

    file, _, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "file field missing", http.StatusBadRequest)
        return
    }
    defer file.Close()

    if err := s.bot.BroadcastVideo(file, "🚨 Possible break-in detected"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}