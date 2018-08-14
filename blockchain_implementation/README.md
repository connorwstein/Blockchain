Blockchain implementation using Nakomoto consensus and a network of containers. 

Could use grpc streams for the miner's listening for transactions and for the normal nodes listening for new blocks.

Normal clients
- Listening for new blocks to add to their chain and broadcasting transactions as desired

Miners
- Listening for transactions which are broadcasted, they compile a bunch to fill up a block, mine the block, 
insert a transaction giving themselves some coin then broadcast this mined block

Seeds
- This is a stable node which clients will have hardcoded as the initial node to connect to when they connect for the first time.
When people connect to this guy and ask for clients he can give you anyone (chooses at random) of the clients that is already connected to him.

DONE
- Simple block with some data
- Downstream blocks invalidated upon tampering

TODO
- Trigger some transactions to be sent - we can use the grpc server running on each node to do that. It can have some RPC which causes a transaction to be generated
- Miners
- Adjust mining difficultly to keep the block mining time constant
- Merkle tree
- Probably all kinds of other stuff I am missing
