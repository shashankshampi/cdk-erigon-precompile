package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

type DeploymentResult struct {
	BlockNumber      uint64 `json:"blockNumber"`
	TransactionHash  string `json:"transactionHash"`
	ContractAddress  string `json:"contractAddress"`
	GasUsed          uint64 `json:"gasUsed"`
	BytecodeSize     int    `json:"bytecodeSize"`
	Status           uint   `json:"status"`
	VerificationPass bool   `json:"verificationPass"`
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

	// Load deployer credentials
	privateKey, fromAddress, err := loadDeployerCredentials()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("🔐 Using deployer address: %s\n", fromAddress.Hex())

	// Get chain ID (override if needed)
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Printf("⚠️  Failed to get chain ID from node, using default: %v", err)
		chainID = big.NewInt(10101) // Default for cdk-erigon devnet
	}
	fmt.Printf("🔗 Network Chain ID: %d\n", chainID)

	// Load contract bytecode
	bytecode, err := os.ReadFile("artifacts/Sha256Wrapper.bin")
	if err != nil {
		log.Fatalf("❌ Failed to read bytecode: %v", err)
	}
	fmt.Println("📦 Bytecode loaded")

	// Deploy contract
	result, err := deployContract(client, privateKey, fromAddress, chainID, string(bytecode))
	if err != nil {
		log.Fatal(err)
	}

	// Verify deployment
	if err := verifyDeployment(client, result); err != nil {
		log.Fatal(err)
	}

	// Save results
	if err := saveResults(result); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n🚀 Deployment successful!")
	fmt.Printf("📝 Results saved to results_stage2.json\n")
	fmt.Printf("📌 Contract Address: %s\n", result.ContractAddress)
}

func loadDeployerCredentials() (*ecdsa.PrivateKey, common.Address, error) {
	privateKeyHex := os.Getenv("DEPLOYER_PRIVATE_KEY")
	if privateKeyHex == "" {
		return nil, common.Address{}, fmt.Errorf("❌ DEPLOYER_PRIVATE_KEY not set in .env")
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("❌ Invalid private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, common.Address{}, fmt.Errorf("❌ Failed to cast public key to ECDSA")
	}

	return privateKey, crypto.PubkeyToAddress(*publicKeyECDSA), nil
}

func deployContract(client *ethclient.Client, privateKey *ecdsa.PrivateKey, fromAddress common.Address, chainID *big.Int, bytecode string) (*DeploymentResult, error) {
	// Get nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to get nonce: %v", err)
	}
	fmt.Printf("🔢 Nonce: %d\n", nonce)

	// Create legacy transaction (TxType 0)
	txData := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(1e9), // 1 Gwei
		Gas:      2_000_000,       // Fixed gas limit as required
		Value:    big.NewInt(0),
		Data:     common.FromHex(strings.TrimSpace(bytecode)),
	}

	tx := types.NewTx(txData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to sign transaction: %v", err)
	}

	// Send transaction
	fmt.Println("📨 Sending deployment transaction...")
	if err := client.SendTransaction(context.Background(), signedTx); err != nil {
		if !strings.Contains(err.Error(), "already known") {
			return nil, fmt.Errorf("❌ Failed to send transaction: %v", err)
		}
		fmt.Println("⚠️  Transaction already known by node")
	}

	// Wait for receipt
	fmt.Println("⏳ Waiting for transaction to be mined...")
	receipt, err := waitForReceipt(client, signedTx.Hash())
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to get receipt: %v", err)
	}

	// Get deployed address
	deployedAddress := crypto.CreateAddress(fromAddress, nonce)

	return &DeploymentResult{
		BlockNumber:     receipt.BlockNumber.Uint64(),
		TransactionHash: signedTx.Hash().Hex(),
		ContractAddress: deployedAddress.Hex(),
		GasUsed:         receipt.GasUsed,
		Status:          uint(receipt.Status),
	}, nil
}

func verifyDeployment(client *ethclient.Client, result *DeploymentResult) error {
	// Check transaction status
	if result.Status != 1 {
		return fmt.Errorf("❌ Contract deployment failed (reverted)! Status: %d, Gas used: %d", result.Status, result.GasUsed)
	}
	fmt.Printf("✅ Transaction mined in block %d\n", result.BlockNumber)

	// Verify contract code exists
	code, err := client.CodeAt(context.Background(), common.HexToAddress(result.ContractAddress), nil)
	if err != nil {
		return fmt.Errorf("❌ Failed to get contract code: %v", err)
	}

	result.BytecodeSize = len(code)
	if result.BytecodeSize == 0 {
		return fmt.Errorf("❌ No contract code found at deployed address %s", result.ContractAddress)
	}

	result.VerificationPass = true
	fmt.Printf("✅ Contract verification passed - Code size: %d bytes\n", result.BytecodeSize)
	return nil
}

func saveResults(result *DeploymentResult) error {
	// Save deployed address
	if err := os.WriteFile("deployed_address.txt", []byte(result.ContractAddress), 0644); err != nil {
		return fmt.Errorf("❌ Failed to save deployed address: %v", err)
	}

	// Save full results
	file, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to marshal results: %v", err)
	}

	if err := os.WriteFile("results_stage2.json", file, 0644); err != nil {
		return fmt.Errorf("❌ Failed to save results: %v", err)
	}
	return nil
}

func waitForReceipt(client *ethclient.Client, txHash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(context.Background(), txHash)
			if err == nil && receipt != nil {
				return receipt, nil
			}
		}
	}
}
