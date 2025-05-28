package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

type Result struct {
	Stage         string `json:"stage"`
	Success       bool   `json:"success"`
	Precompile    string `json:"precompile"`
	Input         string `json:"input"`
	ExpectedHash  string `json:"expected_hash"`
	ReturnedHash  string `json:"returned_hash"`
	Match         bool   `json:"match"`
	Error         string `json:"error,omitempty"`
	Timestamp     string `json:"timestamp"`
	Network       string `json:"network"`
	RPCURL        string `json:"rpc_url"`
	TransactionID string `json:"transaction_id,omitempty"`
}

// Helper function to get pointer to address
func addressPtr(addr common.Address) *common.Address {
	return &addr
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Get configuration from environment
	rpcHost := os.Getenv("RPC_HOST")
	if rpcHost == "" {
		rpcHost = "127.0.0.1" // default
	}

	rpcPort := os.Getenv("RPC_PORT")
	if rpcPort == "" {
		rpcPort = "63311" // default
	}

	rpcURL := fmt.Sprintf("http://%s:%s", rpcHost, rpcPort)
	inputData := "hello world"

	// Initialize result struct
	result := Result{
		Stage:      "Stage 1 - Raw Precompile Invocation",
		Precompile: "0x02", // SHA256 precompile address
		Input:      inputData,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Network:    "cdk-erigon",
		RPCURL:     rpcURL,
	}

	// Connect to client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		result.Error = fmt.Sprintf("Client connection error: %v", err)
		saveResult(result)
		log.Fatalf(result.Error)
	}

	// Verify network
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("Network verification error: %v", err)
		saveResult(result)
		log.Fatalf(result.Error)
	}
	fmt.Printf("Connected to network with ChainID: %d\n", chainID)

	// Prepare input
	input := []byte(result.Input)
	expected := sha256.Sum256(input)
	result.ExpectedHash = fmt.Sprintf("%x", expected)

	// Create call message using the helper function
	msg := ethereum.CallMsg{
		To:   addressPtr(common.HexToAddress(result.Precompile)),
		Data: input,
	}

	// Call precompile
	callResult, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		result.Error = fmt.Sprintf("Precompile call error: %v", err)
		saveResult(result)
		log.Fatalf(result.Error)
	}

	// Process results
	result.ReturnedHash = fmt.Sprintf("%x", callResult)
	result.Match = result.ExpectedHash == result.ReturnedHash
	result.Success = true

	// Print and save results
	fmt.Println("\n=== Precompile Call Results ===")
	fmt.Printf("RPC Endpoint: %s\n", rpcURL)
	fmt.Printf("Precompile Address: %s\n", result.Precompile)
	fmt.Printf("Input: %q\n", input)
	fmt.Printf("Expected SHA256: %s\n", result.ExpectedHash)
	fmt.Printf("Returned SHA256: %s\n", result.ReturnedHash)

	if result.Match {
		fmt.Println("✅ Result matches expected hash")
	} else {
		fmt.Println("❌ Result DOES NOT match expected hash")
	}

	saveResult(result)
}

func saveResult(result Result) {
	file, err := os.Create("results_stage1.json")
	if err != nil {
		log.Fatalf("Failed to create results file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}
	fmt.Println("Results saved to results_stage1.json")
}
