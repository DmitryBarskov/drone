package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func Path() string {
	path, ok := os.LookupEnv("DRONE_CONFIG_FILENAME")
	if !ok {
		return "config.yaml"
	}

	return path
}

//go:embed default_config.yaml
var defaultConfigBytes []byte

type Config struct {
	BadgerPath string `yaml:"badger_path"`

	Features struct {
		SnippetsGenerationEnabled bool `yaml:"snippets_generation_enabled"`
	} `yaml:"features"`

	Tg struct {
		Key               string        `yaml:"api_key" json:"-"` // intentionally hidden from logs
		LongPollerTimeout time.Duration `yaml:"long_poller_timeout"`
		AdminChatID       int64         `yaml:"admin_chat_id"`
	} `yaml:"tg"`

	Rod struct {
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		DownloadsFolder string `yaml:"downloads_folder"`
	} `yaml:"rod"`

	MediaGenerator struct {
		CarbonURL          string `yaml:"carbon_url"`
		RaysoURL           string `yaml:"rayso_url"`
		JavaHighlightURL   string `yaml:"javahighlight_url"`
		UseCarbon          bool   `yaml:"use_carbon"`
		UseRayso           bool   `yaml:"use_rayso"`
		UseJavaHighlight   bool   `yaml:"use_javahighlight"`
		RodDownloadsFolder string `yaml:"rod_downloads_folder"`
	} `yaml:"media_generator"`

	VC struct {
		Domain string `yaml:"domain"`
		Token  string `yaml:"token" json:"-"` // intentionally hidden from logs
	} `yaml:"vc"`

	Boardwhite struct {
		ChatID                   int64 `yaml:"chat_id"`
		LeetCodeThreadID         int   `yaml:"leetcode_thread_id"`
		LeetcodeChickensThreadID int   `yaml:"leetcode_chickens_thread_id"`
		FloodThreadID            int   `yaml:"flood_thread_id"`
	} `yaml:"boardwhite"`

	Leetcode struct {
		Session string `yaml:"session" json:"-"` // intentionally hidden from logs
		CSRF    string `yaml:"csrf" json:"-"`    // intentionally hidden from logs
	} `yaml:"leetcode"`

	LeetcodeDaily struct {
		Cron       string `yaml:"cron"`
		RatingCron string `yaml:"rating_cron"`
	} `yaml:"leetcode_daily"`

	NeetcodeDaily struct {
		Cron       string `yaml:"cron"`
		RatingCron string `yaml:"rating_cron"`
	} `yaml:"neetcode_daily"`

	DailyStickerIDs         []string `yaml:"daily_sticker_ids"`
	DailyChickensStickerIDs []string `yaml:"daily_chickens_sticker_ids"`
	DPStickerID             string   `yaml:"dp_sticker_id"`

	Mocks []struct {
		Username   string   `yaml:"username"`
		Period     string   `yaml:"period"`
		StickerIDs []string `yaml:"sticker_ids"`
	} `yaml:"mocks"`

	GreetingsNewUsersTemplates []string `yaml:"greetings_new_users_templates"`
	GreetingsOldUsersTemplates []string `yaml:"greetings_old_users_templates"`
}

func (cfg Config) String() string {
	b, _ := json.Marshal(&cfg) //nolint:errchkjson // intentionally omitting the error
	return string(b)
}

func Default() (cfg Config, err error) {
	err = yaml.Unmarshal(defaultConfigBytes, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("unmarshal default config: %w", err)
	}

	return cfg, nil
}

func Load(filename string) (Config, error) {
	cfg, err := Default()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", filename, err)
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (cfg Config) validate() error {
	// TODO: 2 phase parsing to be able to use enums
	enabledFlags := []bool{
		cfg.MediaGenerator.UseCarbon,
		cfg.MediaGenerator.UseRayso,
		cfg.MediaGenerator.UseJavaHighlight,
	}
	enabledCount := 0
	for _, flag := range enabledFlags {
		if flag {
			enabledCount++
		}
	}

	if enabledCount != 1 {
		return errors.New("only one of use_carbon, use_rayso, and use_javahighlight should be enabled")
	}

	return nil
}
