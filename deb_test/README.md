## How to make Anduschain DEB Network

#### Test Fairnode KEY PASS = '11111'

1. Running Mongodb ( mongoDB service :  mongodb://localhost/Anduschain_14288642 | PORT : 27017 )
2. Running fairnode addChainConfig
    ```$xslt
        $ ./prepare.sh
    ```
3. Running goreman start
    ```$xslt
        $ goreman start
    ```
4. Attach Console
    ```$xslt
        cosole A > $ godaon attach http://localhost:8545
        cosole B > $ godaon attach http://localhost:8546
        cosole C > $ godaon attach http://localhost:8547
    ```
5. Start Mining ( type each console this )
    ```$xslt
        $ godaon> miner.start()
    ```

#### Before runnign AndusChain
1. [Set goreman setting](https://github.com/mattn/goreman)
2. [Running MongoDB](https://hub.docker.com/_/mongo)