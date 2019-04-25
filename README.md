## Go Anduschain

Official golang implementation of the andusdeb protocol.

## System requirements

```
it is recommended to have at least 4 GB of memory available when running anduschain.
OS : Linux(CentOS, Ubuntu), Mac, Windows
CPU : least 4 CORE CPU
```

## Building the source

Building geth requires both a Go (version 1.10 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make geth

or, to build the full suite of utilities:

    make all
    
## A Full node on the Anduschain test network

You need to get DAON coin for mining in Anduschain network.
We're getting ready to deliver the test coin.
You can join whenever you want.

```
$ geth --testnet console
> geth version : Geth/v0.6.7-anduschain-unstable-3c7dd541
```

## Issue report
You will report the issue, using geth. It will connect github issue site. 
```
$ geth bug
```

## License

The go-anduschain library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-anduschain binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
