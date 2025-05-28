package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	client, err := ethclient.Dial("http://127.0.0.1:63311")
	if err != nil {
		log.Fatalf("Failed to connect to client: %v", err)
	}

	// Read deployed contract address
	addrBytes, err := os.ReadFile("deployed_address.txt")
	if err != nil {
		log.Fatalf("❌ Failed to read deployed address: %v", err)
	}
	deployedAddrStr := strings.TrimSpace(string(addrBytes))
	wrapperAddress := common.HexToAddress(deployedAddrStr)

	// Verify contract is deployed (code exists)
	code, err := client.CodeAt(context.Background(), wrapperAddress, nil)
	if err != nil {
		log.Fatalf("Failed to get contract code: %v", err)
	}
	if len(code) == 0 {
		log.Fatalf("No contract code found at address %s", wrapperAddress.Hex())
	}
	fmt.Printf("✅ Contract found at %s (code size: %d bytes)\n", wrapperAddress.Hex(), len(code))

	// ABI of your wrapper contract
	abiBytes, err := os.ReadFile("artifacts/Sha256Wrapper.abi")
	if err != nil {
		log.Fatalf("Failed to read ABI: %v", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		log.Fatalf("Failed to parse ABI: %v", err)
	}

	input := []byte("hello world")

	// Local SHA256 verification
	expected := sha256.Sum256(input)
	fmt.Printf("Expected SHA256(\"hello world\"): %x\n", expected)

	// Pack the function call data
	callData, err := parsedABI.Pack("sha256Hash", input)
	if err != nil {
		log.Fatalf("Failed to pack ABI call: %v", err)
	}

	// Prepare the call message
	msg := ethereum.CallMsg{
		To:   &wrapperAddress,
		Data: callData,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Fatalf("CallContract failed: %v", err)
	}

	// Debug print raw returned bytes
	fmt.Printf("Raw returned bytes: %x\n", result)

	// ABI unpack the result
	unpacked, err := parsedABI.Unpack("sha256Hash", result)
	if err != nil {
		log.Fatalf("Failed to unpack contract call result: %v", err)
	}

	hashBytes, ok := unpacked[0].([32]byte)
	if !ok {
		log.Fatalf("Unexpected type in unpacked result: %T", unpacked[0])
	}

	fmt.Printf("SHA256 from wrapper: %x\n", hashBytes)

	// Compare with expected
	if hashBytes != expected {
		log.Println("❌ Mismatch between wrapper and expected hash.")
	} else {
		log.Println("✅ Hash matches expected SHA256 value.")
	}
}
