package merry

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/catalogfi/blockchain"
	"github.com/catalogfi/blockchain/localnet"
	"github.com/ethereum/go-ethereum/common"
)

func (m *Merry) Fund(to string) error {
	if !m.Running {
		return fmt.Errorf("merry is not running")
	}

	if _, err := btcutil.DecodeAddress(to, &chaincfg.RegressionNetParams); err == nil {
		return fundBTC(to)
	}

	hexRegex := regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	if hexRegex.MatchString(to) {
		return fundStarknet(to)
	}
	
	if len(to) == 42 {
		to = to[2:]
	}
	if _, err := hex.DecodeString(to); err == nil {
		return fundEVM(to)
	}

	return fmt.Errorf("Invalid address %s. Expected a valid ethereum, starknet or bitcoin regtest address", to)
}

func fundEVM(to string) error {
	ethAmount, _ := new(big.Int).SetString("1000000000000000000", 10)
	seedAmount, _ := new(big.Int).SetString("1000000000000000000", 10)

	wbtcAmount, _ := new(big.Int).SetString("100000000", 10)
	wallet, err := localnet.EVMWallet(0)
	if err != nil {
		return err
	}
	tx, err := wallet.Send(context.Background(), localnet.ETH(), common.HexToAddress(to), ethAmount)
	if err != nil {
		return fmt.Errorf("failed to send eth: %v", err)
	}

	ethereumWBTCAsset := blockchain.NewERC20(blockchain.NewEvmChain(blockchain.EthereumLocalnet), common.HexToAddress("0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512"), common.HexToAddress("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"))

	fmt.Printf("Successfully sent %v ETH on Ethereum Localnet at: http://localhost:5100/tx/%s\n", ethAmount, tx.Hash().Hex())
	tx2, err := wallet.Send(context.Background(), ethereumWBTCAsset, common.HexToAddress(to), wbtcAmount)
	if err != nil {
		return fmt.Errorf("failed to send eth: %v", err)
	}
	fmt.Printf("Successfully sent %v WBTC on Ethereum Localnet at: http://localhost:5100/tx/%s\n", wbtcAmount, tx2.Hash().Hex())
	tx3, err := wallet.Send(context.Background(), localnet.ArbitrumETH(), common.HexToAddress(to), ethAmount)
	if err != nil {
		return fmt.Errorf("failed to send eth: %v", err)
	}

	fmt.Printf("Successfully sent %v ETH on Arbitrum Localnet at: http://localhost:5101/tx/%s\n", wbtcAmount, tx3.Hash().Hex())
	arbWBTCAsset := blockchain.NewERC20(blockchain.NewEvmChain(blockchain.ArbitrumLocalnet), common.HexToAddress("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"), common.HexToAddress("0x0165878A594ca255338adfa4d48449f69242Eb8F"))
	tx4, err := wallet.Send(context.Background(), arbWBTCAsset, common.HexToAddress(to), wbtcAmount)
	if err != nil {
		return fmt.Errorf("failed to send eth: %v", err)
	}
	fmt.Printf("Successfully sent %v WBTC on Arbitrum Localnet at: http://localhost:5101/tx/%s\n", wbtcAmount, tx4.Hash().Hex())

	arbSeedAsset := blockchain.NewERC20(blockchain.NewEvmChain(blockchain.ArbitrumLocalnet), common.HexToAddress("0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512"), common.HexToAddress("0x5FC8d32690cc91D4c39d9d3abcBD16989F875707"))
	tx5, err := wallet.Send(context.Background(), arbSeedAsset, common.HexToAddress(to), wbtcAmount)
	if err != nil {
		return fmt.Errorf("failed to send eth: %v", err)
	}

	fmt.Printf("Successfully sent %v SEED on Arbitrum Localnet at: http://localhost:5101/tx/%s\n", seedAmount, tx5.Hash().Hex())
	return nil
}

func fundBTC(to string) error {
	payload, err := json.Marshal(map[string]string{
		"address": to,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal address: %v", err)
	}

	res, err := http.Post("http://127.0.0.1:3000/faucet", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to get funds from faucet: %v", err)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New(string(data))
	}
	var dat map[string]string
	if err := json.Unmarshal([]byte(data), &dat); err != nil {
		return errors.New("internal error, please try again")
	}
	if dat["txId"] == "" {
		return errors.New("not successful")
	}
	fmt.Println("Successfully submitted at http://localhost:5050/tx/" + dat["txId"])
	return nil
}

func fundStarknet(to string) error {
	mintAmount, _ := new(big.Int).SetString("1000000000000000000", 10)

	payload, err := json.Marshal(map[string]any{
		"address": to,
		"amount":  mintAmount,
		"unit":    "FRI",
	})
	
	if err != nil {
		return fmt.Errorf("failed to marshal address: %v", err)
	}
	
	res, err := http.Post("http://localhost:8547/mint", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	
	if res.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}
	
	var data map[string]string
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return errors.New("internal error, please try again")
	}
	
	if data["tx_hash"] == "" {
		return errors.New("error funding address")
	}
	
	fmt.Printf("Successfully funded address. TxHash: %s. New Balance: %s %s.\n", data["tx_hash"], data["new_balance"], data["unit"])
	return nil
}