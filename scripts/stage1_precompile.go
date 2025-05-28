package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

func main() {
	client, err := ethclient.Dial("http://127.0.0.1:63311")
	if err != nil {
		log.Fatalf("Client error: %v", err)
	}

	input := []byte("hello world")
	precompile := common.HexToAddress("0x02")
	msg := ethereum.CallMsg{
		To:   &precompile,
		Data: input,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Fatalf("CallContract error: %v", err)
	}

	expected := sha256.Sum256(input)
	fmt.Printf("Expected: %x\n", expected)
	fmt.Printf("Returned: %x\n", result)
}
