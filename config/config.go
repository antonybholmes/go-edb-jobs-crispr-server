package config

import (
	"os"
	"sync"

	"github.com/antonybholmes/go-sys/log"

	"github.com/antonybholmes/go-sys"
	"go.yaml.in/yaml/v4"
)

type AppConfig struct {
	Redis struct {
		Addr     string `yaml:"-"`
		Password string `yaml:"-"`
	} `yaml:"-"`
	App struct {
		Email struct {
			From string `yaml:"from"`
		} `yaml:"email"`
		DoNotReply string          `yaml:"doNotReply"`
		Version    sys.VersionInfo `yaml:"-"`
	} `yaml:"app"`
	AWS struct {
		S3 struct {
			Buckets struct {
				Jobs string `yaml:"-"`
			} `yaml:"-"`
		} `yaml:"s3"`
		SQS struct {
			QueueURLs struct {
				Jobs struct {
					Crispr    string `yaml:"-"`
					Completed string `yaml:"-"`
				} `yaml:"-"`
			} `yaml:"-"`
			Threads      int `yaml:"threads"`
			WaitTimeSecs int `yaml:"waitTimeSecs"`
		} `yaml:"sqs"`
	} `yaml:"aws"`
}

const (
	AppName = "Experiments Crispr Job Server"
	AppId   = "edb-crispr-job-server"
	//EmailFrom      = "Experiments Team <donotreply@rdf-lab.org>"
	Copyright  = "Copyright (C) 2026 Antony Holmes. All rights reserved."
	ConfigFile = "config/app.yaml"
)

var (
	once sync.Once
	cfg  *AppConfig

	Updated string
	//RedisAddr     string
	//RedisPassword string

	UrlSignIn      string
	UrlVerifyEmail string
	//SqsQueueURL    *string
)

func loadYaml() AppConfig {
	data, err := os.ReadFile(ConfigFile)

	if err != nil {
		log.Fatal().Msgf("failed to read config file: %v", err)
	}

	var cfg AppConfig

	if err := yaml.Load(data, &cfg); err != nil {
		log.Fatal().Msgf("failed to parse config file: %v", err)
	}

	return cfg
}

func load() *AppConfig {

	cfg := loadYaml()
	//env.Load("version.env")

	//Version = os.Getenv("VERSION")
	//Updated = os.Getenv("UPDATED")
	//Copyright = os.Getenv("COPYRIGHT")
	UrlVerifyEmail = os.Getenv("URL_VERIFY_EMAIL")

	//RedisAddr = os.Getenv("REDIS_ADDR")
	//RedisPassword = os.Getenv("REDIS_PASSWORD")

	//SqsQueueURL = aws.String(os.Getenv("AWS_SQS_QUEUE_URL"))

	UrlSignIn = os.Getenv("URL_SIGN_IN")

	cfg.Redis.Addr = os.Getenv("REDIS_ADDR")
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")

	cfg.AWS.S3.Buckets.Jobs = os.Getenv("AWS_S3_BUCKET_JOBS")

	cfg.AWS.SQS.QueueURLs.Jobs.Crispr = os.Getenv("AWS_SQS_QUEUE_JOBS_CRISPR_URL")
	cfg.AWS.SQS.QueueURLs.Jobs.Completed = os.Getenv("AWS_SQS_QUEUE_JOBS_COMPLETED_URL")
	cfg.App.Version = sys.Must(sys.LoadVersionInfo("version.json"))

	return &cfg
}

func Instance() *AppConfig {
	once.Do(func() {
		cfg = load()
	})

	return cfg
}
