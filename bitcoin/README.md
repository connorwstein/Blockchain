Bare bones bitcoin implementation using a network of containers and protobuf/grpc. Supports arbitrary transaction
size with the UTXO model and ECC signing of transactions. 

###### Steps to use
Install docker and docker-compose if you don't have it.
~~~
brew install jq
docker build -t bitcoin_node
docker-compose up
~~~
Save all the ips of each container to a file
~~~
./get_ips.sh
cat networks.txt 
~~~

Start "bitcoin" on each node (from different shells)
~~~
docker-exec -it miner2 bash
./build.sh
./bitcoin &> /tmp/log &
docker-exec -it alice bash
./bitcoin &> /tmp/log &
docker-exec -it bob bash
./bitcoin &> /tmp/log &
docker-exec -it connor bash
./bitcoin &> /tmp/log &
docker-exec -it miner1 bash
./bitcoin &> /tmp/log &
~~~
Now they should peer with whoever they are actually connected to, forming a network:
```
   miner2 -- Alice -- bob 
               | 
            Connor 
               | 
            miner1 
```

Now on any node you can run the following commands
~~~
go run client/client.go new -name=<name> // Create a wallet, do this first!
go run client/client.go wallet -get=address // Get address of wallet
go run client/client.go wallet -get=balance // Get balance of wallet
go run client/client.go mine -action=<start|stop> // Start/stop mining 
go run client/client.go state -get=blocks // Show the blockchain in order 
go run client/client.go state -get=transactions // Show the mempool of transactions on the node
go run client/client.go send -dest=<address> -amount=<amount> // Create a transaction (node a miner needs to be running for it to go through and have balances updated), supports sending arbitrary amounts 
~~~

Example

terminal1: 
```
docker-compose up
```

terminal2: 
```
docker exec -it miner2 bash
           ./build.sh 
           ./bitcoin &> /tmp/log &
           go run client/client.go new -name=miner // Create a wallet
           5707522640979762790628017781145019541280723040962911387402074939254137511372684325613897519128876859788691640375132698542554103264688342511383374429617831
           go run client/client.go mine -action=start // Start mining
           go run client/client.go wallet -get=balance // Periodically check this to watch the balance increase as blocks are solved
           go run client/client.go state -get=blocks // Watch the chain of blocks form (only miner rewards for now)
           // send alice 8 coin
           go run client/client.go send -dest=9771452820233697997201961640375653685974542179741898164207255486067654332003237375903780441578847636490738792595470735310844332696420050890272936749543180 -amount=8 
           // send connor 8 coin, notice how even though we are not directly connected the transaction and subsequent block will get flooded
           // via alice
           go run client/client.go send -dest=69394599743382830167289996945215716749186388199894748240719481065970989669990104271262261088404866051401833414734376420718944618704234778172265777392571220 -amount=8 
```

terminal3: 
```
docker exec -it alice bash
           ./build.sh
           ./bitcoin &> /tmp/log &
           go run client/client.go new -name=alice // Create a wallet
           9771452820233697997201961640375653685974542179741898164207255486067654332003237375903780441578847636490738792595470735310844332696420050890272936749543180
           // after miner2 sends us coin, poll get balance until we see it appear
           go run client/client.go wallet -get=balance 
           8
           // Now we can spend our 8 coin however we want, miner2 will validate it and broadcast the block 
```

terminal4:
```
docker exec -it connor bash
           ./builds.sh
           ./bitcoin &> /tmp/log &
           go run client/client.go new -name=connor // Create a wallet
           69394599743382830167289996945215716749186388199894748240719481065970989669990104271262261088404866051401833414734376420718944618704234778172265777392571220
           // after miner2 sends us coin, poll get balance until we see it appear
           go run client/client.go wallet -get=balance 
           8
           // Now we can spend our 8 coin however we want, miner2 will validate it and broadcast the block 
```

###### Implementation does not include
- Handling temporary forks and switching to longest chain if for example a secondary chain becomes longer (although the data structures are present to support this - see tipsOfChains)
- Multiple miners at the same time and handling orphans (although relatively easy to add)
- Node syncing to an existing network (could be added the peering code)
- Real bootstrapping
- Adjusting mining difficulty over time
- Scripts to unlock UTXO
- Multiple keys per wallet
- SPV nodes
- Persistence

###### Node functionality
- Listening for new blocks to add to their chain 
- Listening for valid transactions, accumulating them in their mempool
- Validating and then relaying valid transactions
- Upon receiving a new block, removing transactions in that block from their mempool
- Each new block will have a pointer (hash) to the previous block thus telling you where to insert it 

###### Miners
- A full node + creation of new blocks
- Aggregate transactions from the mempool, attempting to mine for mining rewards
- If successful in mining a block, update the mempool and broadcast the new block.
