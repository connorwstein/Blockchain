Bare bones EVM implementation

Currently working:

return.sol/input_return.json
1. Alter return.sol with a specified return value X then solc --bin-runtime return.sol -o . --overwrite
2. Take the byte code in Return.bin-runtime and put it in input.json in contractCode
3. go build 
4. ./evm input_return.json
5. Contract byte code gets executed and value X is left at the address
on the top of the stack 

storage.sol/input_storage.json
- Similar to return except you can alter 0x42 to something else
and observe it in storage 
- You can also observe the require fail and cause a revert if you
dont pass enough in the callValue
