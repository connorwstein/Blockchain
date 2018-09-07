package client

import (
	"net/http"
	"fmt"
	"encoding/json"
	"bytes"
	"io/ioutil"
)

type EthAccountsRequest struct {
	Jsonrpc string  `json:"jsonrpc"` 
	Method string   `json:"method"` 
	Params []string `json:"params"` 
	Id int	  		`json:"id"` 
}

type EthAccountsResponse struct {
	Id int `json:"id"` 
	Jsonrpc string `json:"jsonrpc"`
	Result []string `json:"result"`
}

// Interact with ganach via RPC 
func main() {
	t := EthAccountsRequest{"2.0", "eth_accounts", []string{}, 1}
	b, err := json.Marshal(t)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(b, err)
	resp, err := http.Post("http://localhost:8545", "application/json", bytes.NewReader(b))
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(body)
	if err != nil {
		fmt.Println(err)
		return	
	}
	var r EthAccountsResponse  
	err = json.Unmarshal(body, &r)
	if err != nil {
		fmt.Println(err)
		return	
	}
	fmt.Println(r)
}
