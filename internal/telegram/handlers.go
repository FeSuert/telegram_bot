package telegram

import "home-alarm-bot/internal/alarm"

type Bot struct {
	tg *API
	al *alarm.Client
}

func NewBot(tg *API, al *alarm.Client) *Bot { return &Bot{tg: tg, al: al} }

func (b *Bot) Handle(u Update) {
	if u.Message == nil { return }

	chat := u.Message.Chat.ID
	switch u.Message.Text {
	case "/arm":
		if err := b.al.Arm(); err != nil {
			_ = b.tg.SendMessage(chat, "❌ could not arm: "+err.Error())
			return
		}
		_ = b.tg.SendMessage(chat, "🔒 System Armed")

	case "/disarm":
		if err := b.al.Disarm(); err != nil {
			_ = b.tg.SendMessage(chat, "❌ could not disarm: "+err.Error())
			return
		}
		_ = b.tg.SendMessage(chat, "🔓 System Disarmed")

	case "/status":
		state, err := b.al.Status()
		if err != nil {
			_ = b.tg.SendMessage(chat, "❌ status error: "+err.Error())
			return
		}
		var message string 
		if state == "ARMED" {
			message = "📟 State: 🚨 Armed"
		} else {
			message = "📟 State: 💤 Disarmed" 
		}
		_ = b.tg.SendMessage(chat, message)

	default:
		_ = b.tg.SendMessage(chat, "🤖 unknown command")
	}
}
