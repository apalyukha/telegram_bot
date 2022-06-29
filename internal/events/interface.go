package events

import "github.com/apalyukha/telegram_bot/internal/events/model"

type ProcessEventStrategy interface {
	Process(eventBody []byte) (model.ProcessedEvent, error)
}
