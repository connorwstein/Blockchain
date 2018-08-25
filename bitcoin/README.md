Simplified bitcoin implementation using a network of containers. 

# Steps to use
brew install jq
docker build -t bitcoin_node
docker-compose up
// Save all the ips of each container to a file, the real bootstrapping is not implemented
./get_ips.sh

// Start the "bitcoind" on each node
docker-exec -it miner2 bash
./build.sh && ./bitcoind
docker-exec -it alice bash
./build.sh && ./bitcoind
docker-exec -it bob bash
./build.sh && ./bitcoind
docker-exec -it connor bash
./build.sh && ./bitcoind
docker-exec -it miner1 bash
./build.sh && ./bitcoind
// Now they should peer with whoever they can forming a network like:

   miner2 -- Alice -- bob
               |
             Connor
               |
             miner1

// Now on any node you can run the following commands
go run client/client.go new -name=<name> // Create a wallet 
go run client/client.go wallet -get=address // Get address of wallet
go run client/client.go wallet -get=balance // Get balance of wallet
go run client/client.go mine -action=<start|stop> // Start/stop mining 
go run client/client.go state -get=blocks // Show the blockchain in order 
go run client/client.go state -get=transactions // Show the mempool of transactions on the node
go run client/client.go send -dest=<address> -amount=<amount> // Create a transaction (node a miner needs to be running for it to go through and have balances updated), supports sending arbitrary amounts 

# Implementation does not include
- Handling temporary forks and switching to longest chain if for example a secondary chain becomes longer (although the data structures are present to support this - see tipsOfChains)
- Handling orphans (although relatively easy to add)
- Node syncing to an existing network (could be added the peering code)
- Real bootstrapping
- Adjusting mining difficulty over time
- Scripts to unlock UTXO
- Multiple keys per wallet
- SPV nodes to use merkle root
- Persistence

# Node functionality
- Listening for new blocks to add to their chain 
- Listening for valid transactions, accumulating them in their mempool
- Validating and then relaying valid transactions
- Upon receiving a new block, removing transactions in that block from their mempool
- Each new block will have a pointer (hash) to the previous block thus telling you where to insert it 

# Miners
- A full node + creation of new blocks
- Aggregate transactions from the mempool, attempting to mine for mining rewards
- If successful in mining a block, update the mempool and broadcast the new block
