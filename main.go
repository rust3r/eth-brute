package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type config struct {
	privKey string
	threads int
}

func parseConfig() (*config, error) {
	var cfg config

	flag.StringVar(&cfg.privKey, "pk", "", "Start private key")
	flag.IntVar(&cfg.threads, "threads", runtime.NumCPU(), "Number of threads")
	flag.Parse()

	if len(cfg.privKey) < 64 {
		return nil, fmt.Errorf("private key length must be large then 64: '%s'", cfg.privKey)
	}

	return &cfg, nil
}

func generateNewPrivKey(hex string) string {
	sh := strings.Split(hex, "")
	possible := "0123456789abcdef"

	for i := len(hex) - 1; i >= 0; i-- {
		point := strings.Index(possible, sh[i])
		if point == 15 {
			sh[i] = "0"
		} else {
			sh[i] = string(possible[point+1])
			break
		}
	}
	return strings.Join(sh, "")
}

func generateAddressFromPrivKey(hex string) string {
	privateKey, err := crypto.HexToECDSA(hex)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	return address
}

func checkBalance(data chan string) {
	client, err := ethclient.Dial("http://185.197.160.119:8545")
	if err != nil {
		log.Fatalf("Client: %s\n", err)
	}
	defer client.Close()

	for {
		creds := <-data
		addr := strings.Split(creds, ":")[1]
		account := common.HexToAddress(addr)
		balance, err := client.BalanceAt(context.Background(), account, nil)
		if err != nil {
			log.Fatalf("Check balance: %s %v\n", creds, err)
		}

		if balance.Cmp(big.NewInt(0)) != 0 {
			data := creds + "\n"
			writeToFound(data, "found.txt")
		}
		fmt.Println(addr, balance)
	}
}

// func testCli(address string) {
// 	client, err := ethclient.Dial("http://185.197.160.119:8545")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer client.Close()

// 	account := common.HexToAddress(address)
// 	balance, err := client.BalanceAt(context.Background(), account, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Println(balance)
// }

func writeToFound(text string, path string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0655)
	if err != nil {
		log.Fatalf("Open file: %s %v\n", text, err)
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		log.Fatalf("Write string: %s %v\n", text, err)
	}
}

func main() {
	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	chData := make(chan string)

	for t := 0; t < cfg.threads; t++ {
		go checkBalance(chData)
	}

	pk := cfg.privKey

	for {
		pk = generateNewPrivKey(pk)
		addr := generateAddressFromPrivKey(pk)
		chData <- fmt.Sprintf("%s:%s", pk, addr)
	}

	// testCli("0xDa1bDd3845203d8108aD24cc9374F2D07E2FEA14")
}
