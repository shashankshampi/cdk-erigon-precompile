package main

import (
	"context"
	"crypto/ecdsa"
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

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("❌ Error loading .env file")
	}

	privateKeyHex := os.Getenv("DEPLOYER_PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("❌ DEPLOYER_PRIVATE_KEY not set in .env")
	}

	client, err := ethclient.Dial("http://127.0.0.1:63311")
	if err != nil {
		log.Fatalf("❌ Failed to connect to client: %v", err)
	}
	fmt.Println("✅ Connected to Ethereum node")

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatalf("❌ Invalid private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("❌ Failed to cast public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Println("🔐 Using deployer address:", fromAddress.Hex())

	chainID := big.NewInt(10101) // override incorrect chain ID
	fmt.Println("🔗 Network Chain ID:", chainID)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("❌ Failed to get nonce: %v", err)
	}
	fmt.Println("🔢 Nonce:", nonce)

	// Read bytecode
	bytecode, err := os.ReadFile("artifacts/Sha256Wrapper.bin")
	if err != nil {
		log.Fatalf("❌ Failed to read bytecode: %v", err)
	}
	fmt.Println("📦 Bytecode loaded, deploying contract...")

	// Create the transaction
	txData := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(1e9), // 1 Gwei
		Gas:      30_000_000,      // Increased gas limit
		Value:    big.NewInt(0),
		Data:     common.FromHex(string(bytecode)),
	}

	tx := types.NewTx(txData)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatalf("❌ Failed to sign transaction: %v", err)
	}

	txHash := signedTx.Hash()
	fmt.Println("📨 Transaction hash:", txHash.Hex())

	// Check if tx already exists
	_, err = client.TransactionReceipt(context.Background(), txHash)
	if err == nil {
		fmt.Println("⚠️  Transaction already exists and might be mined. Skipping send.")
		return
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		if strings.Contains(err.Error(), "already known") {
			fmt.Println("⚠️  Transaction already known by the node. Waiting for receipt...")
		} else {
			log.Fatalf("❌ Failed to send transaction: %v", err)
		}
	} else {
		fmt.Println("🚀 Transaction sent, waiting to be mined...")
	}

	// Wait for receipt
	receipt, err := waitForReceipt(client, signedTx.Hash())
	if err != nil {
		log.Fatalf("Failed to get receipt: %v", err)
	}

	// Check transaction status
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalf("❌ Contract deployment failed (reverted)! Status: %d, Gas used: %d", receipt.Status, receipt.GasUsed)
	}
	fmt.Println("✅ Transaction mined in block", receipt.BlockNumber.Uint64())

	// Get the deployed address
	deployedAddress := crypto.CreateAddress(fromAddress, nonce)
	fmt.Println("✅ Contract deployed at:", deployedAddress.Hex())

	// Verify contract code exists
	code, err := client.CodeAt(context.Background(), deployedAddress, nil)
	if err != nil {
		log.Fatalf("Failed to get contract code at deployed address: %v", err)
	}
	if len(code) == 0 {
		log.Fatalf("No contract code found at deployed address %s", deployedAddress.Hex())
	}
	fmt.Printf("✅ Contract code size: %d bytes\n", len(code))

	// Write address to file
	err = os.WriteFile("deployed_address.txt", []byte(deployedAddress.Hex()), 0644)
	if err != nil {
		log.Fatalf("❌ Failed to save deployed address: %v", err)
	}
}

func waitForReceipt(client *ethclient.Client, txHash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
