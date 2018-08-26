Simplified bitcoin implementation using a network of containers and protobuf/grpc. Supports arbitrary transaction
size with the UTXO model and ECC signing of transactions. 

###### Steps to use
~~~
brew install jq
docker build -t bitcoin_node
docker-compose up
~~~
// Save all the ips of each container to a file
~~~
./get_ips.sh
cat networks.txt 
~~~

// Start "bitcoin" on each node (from different shells)
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
// Now they should peer with whoever they are actually connected to, forming a network:
```
   miner2 -- Alice -- bob 
               | 
            Connor 
               | 
            miner1 
```

// Now on any node you can run the following commands
~~~
go run client/client.go new -name=<name> // Create a wallet 
go run client/client.go wallet -get=address // Get address of wallet
go run client/client.go wallet -get=balance // Get balance of wallet
go run client/client.go mine -action=<start|stop> // Start/stop mining 
go run client/client.go state -get=blocks // Show the blockchain in order 
go run client/client.go state -get=transactions // Show the mempool of transactions on the node
go run client/client.go send -dest=<address> -amount=<amount> // Create a transaction (node a miner needs to be running for it to go through and have balances updated), supports sending arbitrary amounts 
~~~

###### Implementation does not include
- Handling temporary forks and switching to longest chain if for example a secondary chain becomes longer (although the data structures are present to support this - see tipsOfChains)
- Handling orphans (although relatively easy to add)
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
