// internal/telegram/handlers.go
package telegram

import (
	"sync"

	"home-alarm-bot/internal/state"
)

type Bot struct {
	tg    *API
	store *state.Store

	mu    sync.RWMutex
	chats map[int64]struct{} // authorised chats
}

func NewBot(tg *API, store *state.Store) *Bot {
	return &Bot{
		tg:    tg,
		store: store,
		chats: make(map[int64]struct{}),
	}
}

// Broadcast sends a message to every known chat.
func (b *Bot) Broadcast(msg string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for id := range b.chats {
		_ = b.tg.SendMessage(id, msg)
	}
}

func (b *Bot) Handle(u Update) {
	if u.Message == nil {
		return
	}
	chatID := u.Message.Chat.ID

	// remember the chat
	b.mu.Lock()
	b.chats[chatID] = struct{}{}
	b.mu.Unlock()

	switch u.Message.Text {
	case "/arm":
		b.store.Set(state.Armed)
		_ = b.tg.SendMessage(chatID, "🔒 System Armed")
	case "/disarm":
		b.store.Set(state.Disarmed)
		_ = b.tg.SendMessage(chatID, "🔓 System Disarmed")
	case "/status":
		if b.store.Get() == state.Armed {
			_ = b.tg.SendMessage(chatID, "📟 State: 🚨 Armed")
		} else {
			_ = b.tg.SendMessage(chatID, "📟 State: 💤 Disarmed")
		}
	default:
		_ = b.tg.SendMessage(chatID, "🤖 unknown command")
	}
}
