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
    
### A Full node on the Ethereum test network

Transitioning towards developers, if you'd like to play around with creating Anduschain contracts, you
almost certainly would like to do that without any real money involved until you get the hang of the
entire system. In other words, instead of attaching to the main network, you want to join the **test**
network with your node, which is fully equivalent to the main network, but with play-Ether only.

```
$ geth --testnet console
```

## License

The go-anduschain library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-anduschain binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
