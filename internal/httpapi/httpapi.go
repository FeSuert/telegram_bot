// internal/httpapi/httpapi.go
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

	return http.ListenAndServe(addr, mux)
}
