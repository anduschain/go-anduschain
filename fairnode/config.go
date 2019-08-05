package fairnode

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

	SSL_path string // for connection mongodb using ssl cert

	//key
	KeyPath string
	KeyPass string

	Port    string // 네트워크 포트
	ChainID uint64
	Debug   bool
	SysLog  bool
	Version string

	Fake bool
}

var (
	DefaultConfig *Config
	Version       = "1.0.2"
)

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
		DBhost:   "localhost",
		DBport:   "27017",
		DBuser:   "",
		SSL_path: "",

		KeyPath: filepath.Join(os.Getenv("HOME"), ".fairnode", "key"),

		Port:    "60002",
		ChainID: 1315,
		Debug:   false,
		SysLog:  false,
		Version: Version, // Fairnode version
	}
}

func (c *Config) GetInfo() (host, port, user, pass, ssl string) {
	return c.DBhost, c.DBport, c.DBuser, c.DBpass, c.SSL_path
}

func SetFairConfig(ctx *cli.Context, keypass, dbpass string) {
	DefaultConfig.KeyPass = keypass
	DefaultConfig.KeyPath = ctx.GlobalString("keypath")
	DefaultConfig.Port = ctx.GlobalString("port")
	DefaultConfig.ChainID = ctx.GlobalUint64("chainID")
	DefaultConfig.Debug = ctx.GlobalBool("debug")

	if ctx.GlobalBool("fake") {
		DefaultConfig.Fake = true
	} else {
		DefaultConfig.DBhost = ctx.GlobalString("dbhost")
		DefaultConfig.DBport = ctx.GlobalString("dbport")
		DefaultConfig.DBuser = ctx.GlobalString("dbuser")
		DefaultConfig.SSL_path = ctx.GlobalString("dbCertPath")
		DefaultConfig.DBpass = dbpass
	}

}
