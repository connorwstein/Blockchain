Bare bones EVM implementation

Currently working:
1. Alter return.sol with a specified return value X then solc --bin-runtime return.sol -o . --overwrite
2. Take the byte code in Return.bin-runtime and put it in input.json in contractCode
3. go build 
4. ./evm input.json
5. Contract byte code gets executed and value X is left at the address
on the top of the stack 
