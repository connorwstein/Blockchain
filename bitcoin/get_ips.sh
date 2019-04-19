#!/bin/bash
rm networks.txt
for (( i=1; i < 5; i++ )); do 
    docker network inspect bitcoin_testnet$i | jq -r 'map(.Containers[].IPv4Address) []' | rev | cut -c4- | rev >> networks.txt
done

