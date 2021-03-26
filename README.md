## Go Anduschain

Official golang implementation of the andusdeb protocol.

## System requirements

```
it is recommended to have at least 8 GB of memory available when running anduschain.
OS : Linux(CentOS, Ubuntu), Mac, Windows
CPU : least 4 CORE CPU
```

## Building the source

go get -d github.com/anduschain/go-anduschain

Building godaon requires both a Go (version 1.10 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make godaon

or, to build the full suite of utilities:

    make all
    
## A Full node on the Anduschain main network

You need to get DAON coin for mining in Anduschain network.
We're getting ready to deliver the test coin.
You can join whenever you want.

```
$ godaon
> godaon version : godaon/v0.7.0-anduschain
```

## A Full node on the Anduschain test network

You need to get DAON coin for mining in Anduschain network.
We're getting ready to deliver the test coin.
You can join whenever you want.

```
$ godaon --testnet
> godaon version : godaon/v0.7.0-anduschain
```

## GoDaon options
```
NAME:
   godaon - the go-anduschain command line interface

   Copyright 2013-2018 The go-anduschain Authors

USAGE:
   godaon [options] command [command options] [arguments...]
   
VERSION:
   0.7.0-anduschain-afff686f
   
COMMANDS:
   account           Manage accounts
   attach            Start an interactive JavaScript environment (connect to node)
   bug               opens a window to report a bug on the geth repo
   console           Start an interactive JavaScript environment
   copydb            Create a local chain from a target chaindata folder
   dbexport          Export blockchain from fairnode db into file
   dump              Dump a specific block from storage
   dumpconfig        Show configuration values
   export            Export blockchain into file
   export-preimages  Export the preimage database into an RLP stream
   exportBlockRlp    ExportBlockRlp blockchain into file
   import            Import a blockchain file
   import-preimages  Import the preimage database from an RLP stream
   init              Bootstrap and initialize a new genesis block
   js                Execute the specified JavaScript files
   license           Display license information
   removedb          Remove blockchain and state databases
   version           Print version numbers
   wallet            Manage Ethereum presale wallets
   help, h           Shows a list of commands or help for one command
   
ANDUS CHAIN OPTIONS:
  --deb      deb network: andus Chain test network [ will deprecated ]
  --testnet  AndusChain test network: pre-configured proof-of-deb test network
  --solo     To make proof-of-deb solo network [ will deprecated ]
  
DEB CLIENT OPTIONS:
  --serverHost value  fairnode connection IP (default: "localhost")
  --serverPort value  fairnode connection Port (default: "60002")
  
DAON OPTIONS:
  --config value                                TOML configuration file
  --datadir "/Users/hakuna/Library/AndusChain"  Data directory for the databases and keystore
  --keystore                                    Directory for the keystore (default = inside the datadir)
  --nousb                                       Disables monitoring for and managing USB hardware wallets
  --networkid value                             Network identifier (integer, 14288640=MainNet(not opening), 14288641=Testnet) (default: 14288640)
  --syncmode "fast"                             Blockchain sync mode ("fast", "full", or "light")
  --gcmode value                                Blockchain garbage collection mode ("full", "archive") (default: "full")
  --ethstats value                              Reporting URL of a ethstats service (nodename:secret@host:port)
  --identity value                              Custom node name
  --lightserv value                             Maximum percentage of time allowed for serving LES requests (0-90) (default: 0)
  --lightpeers value                            Maximum number of LES client peers (default: 100)
  --lightkdf                                    Reduce key-derivation RAM & CPU usage at some expense of KDF strength
  
DEVELOPER CHAIN OPTIONS:
  --dev  Ephemeral proof-of-deb network with a pre-funded developer account, mining enabled
  
TRANSACTION POOL OPTIONS:
  --txpool.locals value        Comma separated accounts to treat as locals (no flush, priority inclusion)
  --txpool.nolocals            Disables price exemptions for locally submitted transactions
  --txpool.journal value       Disk journal for local transaction to survive node restarts (default: "transactions.rlp")
  --txpool.rejournal value     Time interval to regenerate the local transaction journal (default: 1h0m0s)
  --txpool.pricelimit value    Minimum gas price limit to enforce for acceptance into the pool (default: 23809523805524)
  --txpool.pricebump value     Price bump percentage to replace an already existing transaction (default: 10)
  --txpool.accountslots value  Minimum number of executable transaction slots guaranteed per account (default: 80)
  --txpool.globalslots value   Maximum number of executable transaction slots for all accounts (default: 20480)
  --txpool.accountqueue value  Maximum number of non-executable transaction slots permitted per account (default: 320)
  --txpool.globalqueue value   Maximum number of non-executable transaction slots for all accounts (default: 5120)
  --txpool.lifetime value      Maximum amount of time non-executable transaction are queued (default: 3h0m0s)
  
PERFORMANCE TUNING OPTIONS:
  --cache value            Megabytes of memory allocated to internal caching (default: 1024)
  --cache.database value   Percentage of cache memory allowance to use for database io (default: 75)
  --cache.gc value         Percentage of cache memory allowance to use for trie pruning (default: 25)
  --trie-cache-gens value  Number of trie node generations to keep in memory (default: 120)
  
ACCOUNT OPTIONS:
  --unlock value    Comma separated list of accounts to unlock
  --password value  Password file to use for non-interactive password input
  
API AND CONSOLE OPTIONS:
  --rpc                  Enable the HTTP-RPC server
  --rpcaddr value        HTTP-RPC server listening interface (default: "localhost")
  --rpcport value        HTTP-RPC server listening port (default: 8545)
  --rpcapi value         API's offered over the HTTP-RPC interface
  --ws                   Enable the WS-RPC server
  --wsaddr value         WS-RPC server listening interface (default: "localhost")
  --wsport value         WS-RPC server listening port (default: 8546)
  --wsapi value          API's offered over the WS-RPC interface
  --wsorigins value      Origins from which to accept websockets requests
  --ipcdisable           Disable the IPC-RPC server
  --ipcpath              Filename for IPC socket/pipe within the datadir (explicit paths escape it)
  --rpccorsdomain value  Comma separated list of domains from which to accept cross origin requests (browser enforced)
  --rpcvhosts value      Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: "localhost")
  --jspath loadScript    JavaScript root path for loadScript (default: ".")
  --exec value           Execute JavaScript statement
  --preload value        Comma separated list of JavaScript files to preload into the console
  
NETWORKING OPTIONS:
  --bootnodes value     Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)
  --bootnodesv4 value   Comma separated enode URLs for P2P v4 discovery bootstrap (light server, full nodes)
  --bootnodesv5 value   Comma separated enode URLs for P2P v5 discovery bootstrap (light server, light nodes)
  --port value          Network listening port (default: 30305)
  --maxpeers value      Maximum number of network peers (network disabled if set to 0) (default: 25)
  --maxpendpeers value  Maximum number of pending connection attempts (defaults used if set to 0) (default: 0)
  --nat value           NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: "any")
  --nodiscover          Disables the peer discovery mechanism (manual peer addition)
  --v5disc              Enables the experimental RLPx V5 (Topic Discovery) mechanism
  --netrestrict value   Restricts network communication to the given IP networks (CIDR masks)
  --nodekey value       P2P node key file
  --nodekeyhex value    P2P node key as hex (for testing)
  
MINER OPTIONS:
  --mine                             Enable mining
  --miner.threads value              Number of CPU threads to use for mining (default: 0)
  --miner.notify value               Comma separated HTTP URL list to notify of new work packages
  --miner.gasprice "23809523805524"  Minimal gas price for mining a transactions
  --miner.gastarget value            Target gas floor for mined blocks (default: 8000000000000000000)
  --miner.gaslimit value             Target gas ceiling for mined blocks (default: 80000000000)
  --miner.etherbase value            Public address for block mining rewards (default = first account) (default: "0")
  --miner.extradata value            Block extra data set by the miner (default = client version)
  --miner.recommit value             Time interval to recreate the block being mined (default: 3s)
  --miner.noverify                   Disable remote sealing verification
  
GAS PRICE ORACLE OPTIONS:
  --gpoblocks value      Number of recent blocks to check for gas prices (default: 20)
  --gpopercentile value  Suggested gas price is the given percentile of a set of recent transaction gas prices (default: 60)
  
VIRTUAL MACHINE OPTIONS:
  --vmdebug  Record information useful for VM and contract debugging
  
LOGGING AND DEBUGGING OPTIONS:
  --nocompaction            Disables db compaction after import
  --verbosity value         Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail (default: 3)
  --vmodule value           Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)
  --backtrace value         Request a stack trace at a specific logging statement (e.g. "block.go:271")
  --debug                   Prepends log messages with call-site location (file and line number)
  --pprof                   Enable the pprof HTTP server
  --pprofaddr value         pprof HTTP server listening interface (default: "127.0.0.1")
  --pprofport value         pprof HTTP server listening port (default: 6060)
  --memprofilerate value    Turn on memory profiling with the given rate (default: 524288)
  --blockprofilerate value  Turn on block profiling with the given rate (default: 0)
  --cpuprofile value        Write CPU profile to the given file
  --trace value             Write execution trace to the given file
  
EXPORT BLOCKCHAIN FROM DB OPTIONS:
  --usesrv        fairnode database protocol is srv
  --dbuser value  fairnode database user
  --dbhost value  fairnode database Host (default: "localhost")
  --dbname value  fairnode database user (default: "Anduschain")
  --dbopt value   fairnode database options
  
METRICS AND STATS OPTIONS:
  --metrics                          Enable metrics collection and reporting
  --metrics.influxdb                 Enable metrics export/push to an external InfluxDB database
  --metrics.influxdb.endpoint value  InfluxDB API endpoint to report metrics to (default: "http://localhost:8086")
  --metrics.influxdb.database value  InfluxDB database name to push reported metrics to (default: "geth")
  --metrics.influxdb.username value  Username to authorize access to the database (default: "test")
  --metrics.influxdb.password value  Password to authorize access to the database (default: "test")
  --metrics.influxdb.host.tag host   InfluxDB host tag attached to all measurements (default: "localhost")
  
WHISPER (EXPERIMENTAL) OPTIONS:
  --shh                       Enable Whisper
  --shh.maxmessagesize value  Max message size accepted (default: 1048576)
  --shh.pow value             Minimum POW accepted (default: 0.2)
  --shh.restrict-light        Restrict connection between two whisper light clients
  
DEPRECATED OPTIONS:
  --minerthreads value         Number of CPU threads to use for mining (deprecated, use --miner.threads) (default: 0)
  --targetgaslimit value       Target gas floor for mined blocks (deprecated, use --miner.gastarget) (default: 8000000000000000000)
  --gasprice "23809523805524"  Minimal gas price for mining a transactions (deprecated, use --miner.gasprice)
  --etherbase value            Public address for block mining rewards (default = first account, deprecated, use --miner.etherbase) (default: "0")
  --extradata value            Block extra data set by the miner (default = client version, deprecated, use --miner.extradata)
  
MISC OPTIONS:
  --help, -h  show help
  

COPYRIGHT:
   Copyright 2018 The go-anduschain Authors
```

## Issue report
You will report the issue, using geth. It will connect github issue site. 
```
$ godaon bug
```

## License

The go-anduschain library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-anduschain binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
