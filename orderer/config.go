package orderer

import (
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"os"
	"path/filepath"
)

type Config struct {
	Version string
	Port    string
	SubPort string
	ChainID *big.Int
	Debug   bool

	KeyPath string
	KeyPass string
	// DB setting
	UseSRV   bool
	DBhost   string
	DBport   string
	DBuser   string
	DBpass   string
	DBoption string
	SSL_path string // for connection mongodb using ssl cert
}

var (
	DefaultConfig = &Config{
		Version: "0.1",
		Port:    "61001",
		SubPort: "61101",
		ChainID: big.NewInt(1000),
		Debug:   false,

		UseSRV:   false,
		DBhost:   "localhost",
		DBport:   "27017",
		DBuser:   "",
		SSL_path: "",
		DBoption: "",

		KeyPath: filepath.Join(os.Getenv("HOME"), ".orderer", "key"),
	}
)

func (c *Config) GetInfo() (useSRV bool, host, port, user, pass, ssl, option string, chainID *big.Int) {
	return c.UseSRV, c.DBhost, c.DBport, c.DBuser, c.DBpass, c.SSL_path, c.DBoption, c.ChainID
}

func SetOrdererConfig(ctx *cli.Context, keypass string, dbpass string) {
	DefaultConfig.Port = ctx.GlobalString("port")
	DefaultConfig.SubPort = ctx.GlobalString("subport")
	DefaultConfig.Debug = ctx.GlobalBool("debug")

	DefaultConfig.ChainID = new(big.Int).SetUint64(ctx.GlobalUint64("chainID"))

	DefaultConfig.UseSRV = ctx.GlobalBool("usesrv")
	DefaultConfig.DBhost = ctx.GlobalString("dbhost")
	DefaultConfig.DBport = ctx.GlobalString("dbport")
	DefaultConfig.DBuser = ctx.GlobalString("dbuser")
	DefaultConfig.SSL_path = ctx.GlobalString("dbCertPath")
	DefaultConfig.DBoption = ctx.GlobalString("dbOption")
	DefaultConfig.DBpass = dbpass
	DefaultConfig.KeyPath = ctx.GlobalString("keypath")
	DefaultConfig.KeyPass = keypass
}
