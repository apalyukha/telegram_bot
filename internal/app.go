package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apalyukha/telegram_bot/internal/config"
	"github.com/apalyukha/telegram_bot/internal/events"
	"github.com/apalyukha/telegram_bot/internal/events/img"
	"github.com/apalyukha/telegram_bot/internal/events/youtube"
	"github.com/apalyukha/telegram_bot/internal/service/bot"
	"github.com/apalyukha/telegram_bot/pkg/client/mq"
	"github.com/apalyukha/telegram_bot/pkg/client/mq/rabbitmq"
	"github.com/apalyukha/telegram_bot/pkg/logging"
	tb "gopkg.in/telebot.v3"
)

type app struct {
	cfg                    *config.Config
	logger                 *logging.Logger
	producer               mq.Producer
	youtubeProcessStrategy events.ProcessEventStrategy
	imgurProcessStrategy   events.ProcessEventStrategy
	bot                    *tb.Bot
}

type App interface {
	Run()
}

func NewApp(logger *logging.Logger, cfg *config.Config) (App, error) {
	return &app{
		cfg:                    cfg,
		logger:                 logger,
		youtubeProcessStrategy: youtube.NewYouTubeProcessEventStrategy(logger),
		imgurProcessStrategy:   img.NewImgurProcessEventStrategy(logger),
	}, nil
}

func (a *app) Run() {
	bot, err := a.createBot()
	if err != nil {
		return
	}
	a.bot = bot
	a.startConsume()
	a.bot.Start()
}

func (a *app) startConsume() {
	a.logger.Info("start consuming")

	consumer, err := rabbitmq.NewRabbitMQConsumer(rabbitmq.ConsumerConfig{
		BaseConfig: rabbitmq.BaseConfig{
			Host:     a.cfg.RabbitMQ.Host,
			Port:     a.cfg.RabbitMQ.Port,
			Username: a.cfg.RabbitMQ.Username,
			Password: a.cfg.RabbitMQ.Password,
		},
		PrefetchCount: a.cfg.RabbitMQ.Consumer.MessagesBufferSize,
	})
	if err != nil {
		a.logger.Fatal(err)
	}
	producer, err := rabbitmq.NewRabbitMQProducer(rabbitmq.ProducerConfig{
		BaseConfig: rabbitmq.BaseConfig{
			Host:     a.cfg.RabbitMQ.Host,
			Port:     a.cfg.RabbitMQ.Port,
			Username: a.cfg.RabbitMQ.Username,
			Password: a.cfg.RabbitMQ.Password,
		},
	})
	if err != nil {
		a.logger.Fatal(err)
	}

	err = consumer.DeclareQueue(a.cfg.RabbitMQ.Consumer.Youtube, true, false, false, nil)
	if err != nil {
		a.logger.Fatal(err)
	}
	ytMessages, err := consumer.Consume(a.cfg.RabbitMQ.Consumer.Youtube)
	if err != nil {
		a.logger.Fatal(err)
	}

	botService := bot.Service{
		Bot:    a.bot,
		Logger: a.logger,
	}

	for i := 0; i < a.cfg.AppConfig.EventWorkers.Youtube; i++ {
		worker := events.NewWorker(i, consumer, a.youtubeProcessStrategy, botService, producer, ytMessages, a.logger)

		go worker.Process()
		a.logger.Infof("YouTube Event Worker #%d started", i)
	}

	err = consumer.DeclareQueue(a.cfg.RabbitMQ.Consumer.Imgur, true, false, false, nil)
	if err != nil {
		a.logger.Fatal(err)
	}
	imgurMessages, err := consumer.Consume(a.cfg.RabbitMQ.Consumer.Imgur)
	if err != nil {
		a.logger.Fatal(err)
	}

	for i := 0; i < a.cfg.AppConfig.EventWorkers.Imgur; i++ {
		worker := events.NewWorker(i, consumer, a.imgurProcessStrategy, botService, producer, imgurMessages, a.logger)

		go worker.Process()
		a.logger.Infof("Imgur Event Worker #%d started", i)
	}

	a.producer = producer
}

func (a *app) createBot() (abot *tb.Bot, botErr error) {
	pref := tb.Settings{
		Token:   a.cfg.Telegram.Token,
		Poller:  &tb.LongPoller{Timeout: 60 * time.Second},
		Verbose: false,
		OnError: a.OnBotError,
	}
	abot, botErr = tb.NewBot(pref)
	if botErr != nil {
		a.logger.Fatal(botErr)
		return
	}

	abot.Handle("/help", func(c tb.Context) error {
		return c.Send(("/yt - find youtube track by name\nupload photo with compressions and get imgur short url"))
	})

	abot.Handle("/yt", func(c tb.Context) error {
		trackName := c.Message().Payload

		request := youtube.SearchTrackRequest{
			RequestID: fmt.Sprintf("%d", c.Sender().ID),
			Name:      trackName,
		}

		marshal, _ := json.Marshal(request)

		err := a.producer.Publish(a.cfg.RabbitMQ.Producer.Youtube, marshal)
		if err != nil {
			return c.Send(fmt.Sprintf("помилка: %s", err.Error()))
		}

		return c.Send("Заявка прийнята")
	})

	abot.Handle(tb.OnPhoto, func(c tb.Context) error {
		// Photos only.
		photo := c.Message().Photo
		file, err := abot.File(&photo.File)
		if err != nil {
			return c.Send("Неможливо завантажити зображення")
		}
		defer file.Close()
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(file)
		if err != nil {
			return c.Send("Неможливо завантажити зображення")
		}

		if buf.Len() > 10_485_760 {
			return c.Send("Ліміт 10MБ")
		}

		request := img.UploadImageRequest{
			RequestID: fmt.Sprintf("%d", c.Sender().ID),
			Photo:     buf.Bytes(),
		}

		marshal, _ := json.Marshal(request)

		err = a.producer.Publish(a.cfg.RabbitMQ.Producer.Imgur, marshal)
		if err != nil {
			return c.Send(fmt.Sprintf("помилка: %s", err.Error()))
		}

		return c.Send("Заявка прийнята")
	})

	return
}

func (a *app) OnBotError(err error, ctx tb.Context) {
	a.logger.Error(err)
}
