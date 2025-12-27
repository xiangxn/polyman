package config

import (
	"fmt"
	"log"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/spf13/viper"

	"github.com/xiangxn/go-polymarket-sdk/headers"
	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
	pmutils "github.com/xiangxn/go-polymarket-sdk/utils"
)

type OrderEngineConfig struct {
	WorkerNum int `mapstructure:"worker_num"`
	QueueSize int `mapstructure:"queue_size"`
}

func NewOrderEngineConfig() *OrderEngineConfig {
	return &OrderEngineConfig{
		WorkerNum: 2,
		QueueSize: 300,
	}
}

type Config struct {
	Encrypt     bool              `mapstructure:"encrypt"`
	OwnerKey    string            `mapstructure:"owner_key"`
	MarketSlug  string            `mapstructure:"market_slug"`
	PmSDK       pm.Config         `mapstructure:"polymarket"`
	OrderEngine OrderEngineConfig `mapstructure:"order_engine"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetDefault("encrypt", false)
	v.SetDefault("market_slug", "eth-updown-15m-")

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// 支持环境变量覆盖
	v.SetEnvPrefix("PM")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = v.ReadInConfig()

	defConfig := pm.DefaultConfig()
	orderEngineConfig := NewOrderEngineConfig()
	cfg := Config{
		Encrypt:     false,
		PmSDK:       *defConfig,
		OrderEngine: *orderEngineConfig,
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	var enc *pmutils.Encryptor

	if cfg.Encrypt {
		fmt.Println("Please enter the startup password:")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal(err)
		}
		enc = pmutils.NewEncryptor(string(bytePassword))
	}

	if v.IsSet("polymarket.clob_creds") {
		var creds headers.ApiKeyCreds
		_ = v.UnmarshalKey("polymarket.clob_creds", &creds)
		if enc != nil {
			k1, err := enc.Decrypt(creds.Key)
			if err != nil {
				return nil, err
			}
			creds.Key = k1
			s1, err := enc.Decrypt(creds.Secret)
			if err != nil {
				return nil, err
			}
			creds.Secret = s1
			p1, err := enc.Decrypt(creds.Passphrase)
			if err != nil {
				return nil, err
			}
			creds.Passphrase = p1
		}
		cfg.PmSDK.Polymarket.CLOBCreds = &creds
	}

	if v.IsSet("polymarket.builder_creds") {
		var signer headers.ApiKeyCreds
		_ = v.UnmarshalKey("polymarket.builder_creds", &signer)
		if enc != nil {
			k1, err := enc.Decrypt(signer.Key)
			if err != nil {
				return nil, err
			}
			signer.Key = k1
			s1, err := enc.Decrypt(signer.Secret)
			if err != nil {
				return nil, err
			}
			signer.Secret = s1
			p1, err := enc.Decrypt(signer.Passphrase)
			if err != nil {
				return nil, err
			}
			signer.Passphrase = p1
		}
		cfg.PmSDK.Polymarket.BuilderCreds = &signer
	}

	if enc != nil {
		pri, err := enc.Decrypt(cfg.OwnerKey)
		if err != nil {
			return nil, err
		}
		cfg.OwnerKey = pri
	}

	return &cfg, nil
}
