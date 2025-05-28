package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

type TestResult struct {
	Input              string `json:"input"`
	ExpectedHash       string `json:"expectedHash"`
	ContractHash       string `json:"contractHash"`
	Match              bool   `json:"match"`
	ContractAddress    string `json:"contractAddress"`
	WrapperCallSuccess bool   `json:"wrapperCallSuccess"`
}

func main() {
	// Load environment variables
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("❌ Error loading .env file")
	}

	// Initialize Ethereum client
	rpcHost := os.Getenv("RPC_HOST")
	rpcPort := os.Getenv("RPC_PORT")
	rpcURL := fmt.Sprintf("http://%s:%s", rpcHost, rpcPort)

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Ethereum node at %s: %v", rpcURL, err)
	}
	defer client.Close()
	fmt.Printf("✅ Connected to Ethereum node at %s\n", rpcURL)

	// Read deployed contract address
	wrapperAddress, err := getDeployedAddress()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("📌 Using contract at: %s\n", wrapperAddress.Hex())

	// Verify contract is deployed
	if err := verifyContract(client, wrapperAddress); err != nil {
		log.Fatal(err)
	}

	// Load contract ABI
	parsedABI, err := loadContractABI()
	if err != nil {
		log.Fatal(err)
	}

	// Test vectors
	testInputs := []string{
		"hello world",
		"",
		"The quick brown fox jumps over the lazy dog",
		"cdk-erigon",
	}

	var results []TestResult

	// Test each input
	for _, input := range testInputs {
		result, err := testHashFunction(client, wrapperAddress, parsedABI, []byte(input))
		if err != nil {
			log.Printf("⚠️  Test failed for input '%s': %v", input, err)
			continue
		}
		results = append(results, *result)
	}

	// Save results
	if err := saveTestResults(results); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n🧪 Test results:")
	for _, res := range results {
		status := "❌"
		if res.Match {
			status = "✅"
		}
		fmt.Printf("%s Input: '%s'\n  Expected: %s\n  Got:      %s\n",
			status, res.Input, res.ExpectedHash, res.ContractHash)
	}
	fmt.Println("\n📝 Results saved to results_stage3.json")
}

func getDeployedAddress() (common.Address, error) {
	addrBytes, err := os.ReadFile("deployed_address.txt")
	if err != nil {
		return common.Address{}, fmt.Errorf("❌ Failed to read deployed address: %v", err)
	}
	deployedAddrStr := strings.TrimSpace(string(addrBytes))
	return common.HexToAddress(deployedAddrStr), nil
}

func verifyContract(client *ethclient.Client, address common.Address) error {
	code, err := client.CodeAt(context.Background(), address, nil)
	if err != nil {
		return fmt.Errorf("❌ Failed to get contract code: %v", err)
	}
	if len(code) == 0 {
		return fmt.Errorf("❌ No contract code found at address %s", address.Hex())
	}
	fmt.Printf("✅ Contract verified (code size: %d bytes)\n", len(code))
	return nil
}

func loadContractABI() (*abi.ABI, error) {
	abiBytes, err := os.ReadFile("artifacts/Sha256Wrapper.abi")
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to read ABI: %v", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to parse ABI: %v", err)
	}

	return &parsedABI, nil
}

func testHashFunction(client *ethclient.Client, wrapperAddress common.Address, parsedABI *abi.ABI, input []byte) (*TestResult, error) {
	// Calculate expected hash locally
	expected := sha256.Sum256(input)
	inputStr := string(input)
	if len(inputStr) > 20 {
		inputStr = inputStr[:20] + "..."
	}

	// Pack the function call
	callData, err := parsedABI.Pack("sha256Hash", input)
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI call: %v", err)
	}

	// Execute the call
	msg := ethereum.CallMsg{
		To:   &wrapperAddress,
		Data: callData,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %v", err)
	}

	// Unpack the result
	unpacked, err := parsedABI.Unpack("sha256Hash", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %v", err)
	}

	hashBytes, ok := unpacked[0].([32]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected return type: %T", unpacked[0])
	}

	return &TestResult{
		Input:              string(input),
		ExpectedHash:       fmt.Sprintf("%x", expected),
		ContractHash:       fmt.Sprintf("%x", hashBytes),
		Match:              hashBytes == expected,
		ContractAddress:    wrapperAddress.Hex(),
		WrapperCallSuccess: true,
	}, nil
}

func saveTestResults(results []TestResult) error {
	file, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %v", err)
	}

	if err := os.WriteFile("results_stage3.json", file, 0644); err != nil {
		return fmt.Errorf("failed to save results: %v", err)
	}
	return nil
}
