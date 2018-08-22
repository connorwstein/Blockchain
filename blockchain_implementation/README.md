Blockchain implementation using Nakomoto consensus and a network of containers. 

Could use grpc streams for the miner's listening for transactions and for the normal nodes listening for new blocks.

Full nodes 
- Listening for new blocks to add to their chain 
- Listening for valid transactions, accumulating them in their mempool
- Validating and then relaying valid transactions
- Upon receiving a new block, removing transactions in that block from their mempool
- Each new block will have a pointer (hash) to the previous block thus telling you where to insert it 

Miners
- A full node + creation of new blocks
- Aggregate transactions from the mempool, attempting to mine for mining rewards
- If successful in mining a block, update the mempool and broadcast the new block

Remaining
- Handling temporary forks and switching to longest chain if for example a secondary chain becomes longer
- Handling orphans (out of order blocks)
- Handling change (spending only part of a previous UTXO)
- Spending validation
- Figure out how to write automated tests
- Figure out how to share a library between client and server for things like hashing a block, stringifying a block
- I'm sure this code is atrocious, need to clean it up while reading effective go
- Nice to have handle peers dying

TO SKIP
- Real bootstrapping
- Adjusting mining difficulty
- Scripts to unlock UTXO
- Wallets and multiple accounts
- SPV nodes
- Persistence
