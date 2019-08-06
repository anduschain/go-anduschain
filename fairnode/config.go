package fairnode

import (
	"github.com/anduschain/go-anduschain/params"
	"gopkg.in/urfave/cli.v1"
	"math/big"
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
	ChainID *big.Int
	Debug   bool
	SysLog  bool
	Version string

	Memorydb bool
}

var (
	Version       = "1.0.2"
	DefaultConfig = Config{
		DBhost:   "localhost",
		DBport:   "27017",
		DBuser:   "",
		SSL_path: "",

		KeyPath: filepath.Join(os.Getenv("HOME"), ".fairnode", "key"),

		Port:    "60002",
		Debug:   false,
		Version: Version, // Fairnode version
	}
)

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

func (c *Config) GetInfo() (host, port, user, pass, ssl string, chainID *big.Int) {
	return c.DBhost, c.DBport, c.DBuser, c.DBpass, c.SSL_path, c.ChainID
}

func SetFairConfig(ctx *cli.Context, keypass, dbpass string) {
	DefaultConfig.KeyPass = keypass
	DefaultConfig.KeyPath = ctx.GlobalString("keypath")
	DefaultConfig.Port = ctx.GlobalString("port")
	DefaultConfig.Debug = ctx.GlobalBool("debug")

	if ctx.GlobalBool("mainnet") {
		DefaultConfig.ChainID = params.MAIN_NETWORK
	} else if ctx.GlobalBool("testnet") {
		DefaultConfig.ChainID = params.TEST_NETWORK
	} else {
		DefaultConfig.ChainID = new(big.Int).SetUint64(ctx.GlobalUint64("chainID"))
	}

	if ctx.GlobalBool("memorydb") {
		DefaultConfig.Memorydb = true
	} else {
		DefaultConfig.DBhost = ctx.GlobalString("dbhost")
		DefaultConfig.DBport = ctx.GlobalString("dbport")
		DefaultConfig.DBuser = ctx.GlobalString("dbuser")
		DefaultConfig.SSL_path = ctx.GlobalString("dbCertPath")
		DefaultConfig.DBpass = dbpass
	}
}
