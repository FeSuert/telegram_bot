package telegram

import (
	"strings"
	"sync"

	alarmPkg "home-alarm-bot/internal/alarm"
	"home-alarm-bot/internal/state"
)

type Bot struct {
    tg    *API
    store *state.Store
    alarm *alarmPkg.Client

    mu    sync.RWMutex
    chats map[int64]struct{}
}

func NewBot(tg *API, store *state.Store, alarm *alarmPkg.Client) *Bot {
    return &Bot{tg: tg, store: store, alarm: alarm, chats: make(map[int64]struct{})}
}

func (b *Bot) Handle(u Update) {
    if u.Message == nil {
        return
    }
    chatID := u.Message.Chat.ID

    // remember chat
    b.mu.Lock()
    b.chats[chatID] = struct{}{}
    b.mu.Unlock()

    txt := strings.TrimSpace(u.Message.Text)

    switch {
    /* ----------- normal state commands ----------- */
    case txt == "/arm":
        if err := b.alarm.Arm(); err != nil {
            _ = b.tg.SendMessage(chatID, "❌ "+err.Error())
            return
        }
        b.store.Set(state.Armed)
        _ = b.tg.SendMessage(chatID, "🔒 System Armed")

    case txt == "/disarm":
        if err := b.alarm.Disarm(); err != nil {
            _ = b.tg.SendMessage(chatID, "❌ "+err.Error())
            return
        }
        b.store.Set(state.Disarmed)
        _ = b.tg.SendMessage(chatID, "🔓 System Disarmed")

    case txt == "/status":
        st, err := b.alarm.Status()
        if err != nil {
            _ = b.tg.SendMessage(chatID, "❌ "+err.Error())
            return
        }
        if st == "ARMED" {
            _ = b.tg.SendMessage(chatID, "📟 State: 🚨 Armed")
        } else {
            _ = b.tg.SendMessage(chatID, "📟 State: 💤 Disarmed")
        }

    /* --------------- change pin ------------------ */
    case strings.HasPrefix(txt, "/change_pin"):
        parts := strings.Fields(txt) // "/change_pin 1234" -> [" /change_pin", "1234"]
        if len(parts) != 2 {
            _ = b.tg.SendMessage(chatID, "Usage: /change_pin 1234")
            return
        }
        pin := parts[1]
        if err := b.alarm.ChangePIN(pin); err != nil {
            _ = b.tg.SendMessage(chatID, "❌ "+err.Error())
            return
        }
        _ = b.tg.SendMessage(chatID, "✅ PIN changed")

    /* -------------- unknown command -------------- */
    default:
        _ = b.tg.SendMessage(chatID, "🤖 unknown command")
    }
}


func (b *Bot) Broadcast(msg string) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for id := range b.chats {
        _ = b.tg.SendMessage(id, msg)
    }
}
