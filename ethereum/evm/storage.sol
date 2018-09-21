pragma solidity ^0.4.0;

contract Storage {
    uint a = 0;

    function setA(uint b) public payable {
		require(msg.value > 10);
        a = b + 0x42;
    }
}
