package backend

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
}

var DefaultConfig = Config{
	FairNodeDir: "",
	DBhost:      "localhost",
	DBport:      "27017",
	DBuser:      "",

	KeyPath: filepath.Join(os.Getenv("HOME"), ".fairnode", "key"),

	Port:    "60002",
	NAT:     "any",
	ChainID: 1,
	Epoch:   100,
}

func init() {
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

func SetFairConfig(ctx *cli.Context, keypass, dbpass string) {
	DefaultConfig.DBhost = ctx.GlobalString("dbhost")
	DefaultConfig.DBport = ctx.GlobalString("dbport")
	DefaultConfig.KeyPath = ctx.GlobalString("keypath")
	DefaultConfig.Port = ctx.GlobalString("port")
	DefaultConfig.NAT = ctx.GlobalString("nat")
	DefaultConfig.ChainID = ctx.GlobalInt64("chainID")
	DefaultConfig.Epoch = ctx.GlobalInt64("epoch")

	DefaultConfig.KeyPass = keypass
	DefaultConfig.DBpass = dbpass
}
