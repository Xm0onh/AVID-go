package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

type ETHTransaction struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Value    float64 `json:"value"`
	GasPrice float64 `json:"gas_price"`
	Nonce    uint64  `json:"nonce"`
}

type BTCTransaction struct {
	TxID string  `json:"txid"`
	Vin  []Vin   `json:"vin"`
	Vout []Vout  `json:"vout"`
	Fee  float64 `json:"fee"`
}

type Vin struct {
	TxID string `json:"txid"`
	Vout uint32 `json:"vout"`
}

type Vout struct {
	Value        float64 `json:"value"`
	ScriptPubKey string  `json:"scriptPubKey"`
}

func randomETHAddress(rng *mrand.Rand) string {
	return common.HexToAddress(fmt.Sprintf("0x%x", rng.Uint64())).Hex()
}

func randomBTCAddress() string {
	address := make([]byte, 20)
	if _, err := rand.Read(address); err != nil {
		panic("failed to read random bytes for address: " + err.Error())
	}
	return fmt.Sprintf("1%s", hex.EncodeToString(address))
}

func randomETHTransaction(rng *mrand.Rand) ETHTransaction {
	return ETHTransaction{
		From:     randomETHAddress(rng),
		To:       randomETHAddress(rng),
		Value:    rng.Float64() * 100,
		GasPrice: rng.Float64() * 100,
		Nonce:    rng.Uint64(),
	}
}

func randomBTCTransaction(rng *mrand.Rand) BTCTransaction {
	vinCount := rng.Intn(5) + 1
	voutCount := rng.Intn(5) + 1
	vin := make([]Vin, vinCount)
	vout := make([]Vout, voutCount)
	for i := range vin {
		vin[i] = Vin{
			TxID: fmt.Sprintf("%x", rng.Uint64()),
			Vout: uint32(rng.Intn(100)),
		}
	}
	for i := range vout {
		vout[i] = Vout{
			Value:        rng.Float64() * 10,
			ScriptPubKey: randomBTCAddress(),
		}
	}
	return BTCTransaction{
		TxID: fmt.Sprintf("%x", rng.Uint64()),
		Vin:  vin,
		Vout: vout,
		Fee:  rng.Float64(),
	}
}

func main() {
	seed, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		fmt.Println("Error generating random seed:", err)
		return
	}
	rng := mrand.New(mrand.NewSource(seed.Int64()))

	numTransactions := 1_000_000

	ethTransactions := make([]ETHTransaction, numTransactions)
	btcTransactions := make([]BTCTransaction, numTransactions)

	for i := 0; i < numTransactions; i++ {
		ethTransactions[i] = randomETHTransaction(rng)
		btcTransactions[i] = randomBTCTransaction(rng)
	}

	ethFile, err := os.Create("eth_transactions.json")
	if err != nil {
		fmt.Println("Error creating ETH transactions file:", err)
		return
	}
	defer ethFile.Close()

	btcFile, err := os.Create("btc_transactions.json")
	if err != nil {
		fmt.Println("Error creating BTC transactions file:", err)
		return
	}
	defer btcFile.Close()

	ethEncoder := json.NewEncoder(ethFile)
	btcEncoder := json.NewEncoder(btcFile)

	if err := ethEncoder.Encode(ethTransactions); err != nil {
		fmt.Println("Error encoding ETH transactions to JSON:", err)
		return
	}

	if err := btcEncoder.Encode(btcTransactions); err != nil {
		fmt.Println("Error encoding BTC transactions to JSON:", err)
		return
	}

	fmt.Println("Successfully generated and wrote ETH and BTC transactions to files.")
}
