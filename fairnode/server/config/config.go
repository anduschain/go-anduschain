package config

import (
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

type Config struct {
	FairNodeDir string

	// DB setting
	DBhost string
	DBport string
	DBuser string
	DBpass string

	//key
	KeyPath string
	KeyPass string

	Port    string // 네트워크 포트
	NAT     string
	ChainID int64
	Epoch   int64
	Debug   bool
	SysLog  bool
	Version string

	GethVersion string
	Miner       int64
	Fee         int64
}

var DefaultConfig *Config

func init() {
	DefaultConfig = NewConfig()
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.FairNodeDir = filepath.Join(home, "AppData", "FairNode")
	} else {
		DefaultConfig.FairNodeDir = filepath.Join(home, ".fairnode")
	}

}

func NewConfig() *Config {
	return &Config{
		DBhost: "localhost",
		DBport: "27017",
		DBuser: "",

		KeyPath: filepath.Join(os.Getenv("HOME"), ".fairnode", "key"),

		Port:    "60002",
		NAT:     "none",
		ChainID: 3355,
		Debug:   false,
		SysLog:  false,
		Version: "1.0.0", // Fairnode version

		//Mining Config
		GethVersion: "0.6.6",
		Miner:       100,
		Epoch:       100,
		Fee:         6,
	}
}

// Set miner config
func (c *Config) SetMiningConf(miner, epoch, fee int64, version string) {
	c.GethVersion = version
	c.Miner = miner
	c.Epoch = epoch
	c.Fee = fee
}

func SetFairConfig(ctx *cli.Context, keypass, dbpass string) {
	DefaultConfig.DBhost = ctx.GlobalString("dbhost")
	DefaultConfig.DBport = ctx.GlobalString("dbport")
	DefaultConfig.DBuser = ctx.GlobalString("dbuser")
	DefaultConfig.KeyPath = ctx.GlobalString("keypath")
	DefaultConfig.Port = ctx.GlobalString("port")
	DefaultConfig.NAT = ctx.GlobalString("nat")
	DefaultConfig.ChainID = ctx.GlobalInt64("chainID")

	DefaultConfig.Debug = ctx.GlobalBool("debug")
	DefaultConfig.SysLog = ctx.GlobalBool("syslog")

	DefaultConfig.KeyPass = keypass
	DefaultConfig.DBpass = dbpass
}
