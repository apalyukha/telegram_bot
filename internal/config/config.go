package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	IsDebug       *bool `yaml:"is_debug" env:"ST-BOT-IsDebug" env-default:"false"  env-required:"true"`
	IsDevelopment *bool `yaml:"is_development" env:"ST-BOT-IsDevelopment" env-default:"false" env-required:"true"`
	Listen        struct {
		Type   string `yaml:"type" env:"ST-BOT-ListenType" env-default:"port"`
		BindIP string `yaml:"bind_ip" env:"ST-BOT-BindIP" env-default:"localhost"`
		Port   string `yaml:"port" env:"ST-BOT-Port" env-default:"8080"`
	} `yaml:"listen" env-required:"true"`
	Telegram struct {
		Token string `yaml:"token" env:"ST-BOT-TelegramToken" env-required:"true"`
	}
	RabbitMQ struct {
		Host     string `yaml:"host" env:"ST-BOT-RabbitHost" env-required:"true"`
		Port     string `yaml:"port" env:"ST-BOT-RabbitPort" env-required:"true"`
		Username string `yaml:"username" env:"ST-BOT-RabbitUsername" env-required:"true"`
		Password string `yaml:"password" env:"ST-BOT-RabbitPassword" env-required:"true"`
		Consumer struct {
			Youtube            string `yaml:"youtube" env:"ST-BOT-RabbitConsumeYoutube" env-required:"true"`
			Imgur              string `yaml:"imgur" env:"ST-BOT-RabbitConsumeImgur" env-required:"true"`
			MessagesBufferSize int    `yaml:"messages_buff_size" env:"ST-BOT-RabbitConsumerMBS" env-default:"100"`
		} `yaml:"consumer" env-required:"true"`
		Producer struct {
			Youtube string `yaml:"youtube" env:"ST-BOT-RabbitProducerYoutube" env-required:"true"`
			Imgur   string `yaml:"imgur" env:"ST-BOT-RabbitProducerImgur" env-required:"true"`
		} `yaml:"producer" env-required:"true"`
	}
	AppConfig AppConfig `yaml:"app" env-required:"true"`
}

type AppConfig struct {
	EventWorkers struct {
		Youtube int `yaml:"youtube" env:"ST-BOT-EventWorksYT" env-default:"3" env-required:"true"`
		Imgur   int `yaml:"imgur" env:"ST-BOT-EventWorksImgur" env-default:"3" env-required:"true"`
	} `yaml:"event_workers"`
	LogLevel string `yaml:"log_level" env:"ST-BOT-LogLevel" env-default:"error" env-required:"true"`
}

var instance *Config
var once sync.Once

func GetConfig(path string) *Config {
	once.Do(func() {
		log.Printf("read application config in path %s", path)

		instance = &Config{}

		if err := cleanenv.ReadConfig(path, instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			log.Print(help)
			log.Fatal(err)
		}
	})
	return instance
}
