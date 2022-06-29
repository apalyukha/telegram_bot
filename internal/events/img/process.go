package img

import (
	"encoding/json"
	"fmt"

	"github.com/apalyukha/telegram_bot/internal/events"
	"github.com/apalyukha/telegram_bot/internal/events/model"
	"github.com/apalyukha/telegram_bot/pkg/logging"
)

type img struct {
	logger *logging.Logger
}

func NewImgurProcessEventStrategy(logger *logging.Logger) events.ProcessEventStrategy {
	return &img{
		logger: logger,
	}
}

func (p *img) Process(eventBody []byte) (response model.ProcessedEvent, err error) {
	event := UploadImageResponse{}
	if err = json.Unmarshal(eventBody, &event); err != nil {
		return response, fmt.Errorf("failed to unmarshal event due to error %v", err)
	}
	var eventErr error
	if event.Meta.Error != nil {
		eventErr = fmt.Errorf(*event.Meta.Error)
	}
	return model.ProcessedEvent{
		RequestID: event.Meta.RequestID,
		Message:   event.Data.URL,
		Err:       eventErr,
	}, nil
}
