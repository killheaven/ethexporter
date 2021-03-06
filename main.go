package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	allWatching []*Watching
	port        string
	updates     string
	eth         *ethclient.Client
)

type Watching struct {
	Name    string
	Address string
	Balance string
}

//
// Connect to geth server
func ConnectionToGeth(url string) error {
	var err error
	eth, err = ethclient.Dial(url)
	return err
}

//
// Fetch ETH balance from Geth server
func GetEthBalance(address string) *big.Float {
	balance, err := eth.BalanceAt(context.TODO(), common.HexToAddress(address), nil)
	if err != nil {
		fmt.Printf("Error fetching ETH Balance for address: %v", address)
	}
	return ToEther(balance)
}

//
// CONVERTS WEI TO ETH
func ToEther(o *big.Int) *big.Float {
	pul, int := big.NewFloat(0), big.NewFloat(0)
	int.SetInt(o)
	pul.Mul(big.NewFloat(0.000000000000000001), int)
	return pul
}

//
// HTTP response handler for /metrics
func MetricsHttp(w http.ResponseWriter, r *http.Request) {
	var allOut []string
	for _, v := range allWatching {
		allOut = append(allOut, fmt.Sprintf("eth_balance{name=\"%v\",address=\"%v\"} %v", v.Name, v.Address, v.Balance))
	}
	fmt.Fprintln(w, strings.Join(allOut, "\n"))
}

//
// Open the addresses.txt file (name:address)
func OpenAddresses(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		object := strings.Split(scanner.Text(), ":")
		if common.IsHexAddress(object[1]) {
			w := &Watching{
				Name:    object[0],
				Address: object[1],
			}
			allWatching = append(allWatching, w)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return err
}

func main() {
	gethUrl := os.Getenv("GETH")
	port = os.Getenv("PORT")

	err := OpenAddresses("addresses.txt")
	if err != nil {
		panic(err)
	}

	err = ConnectionToGeth(gethUrl)
	if err != nil {
		panic(err)
	}

	// check address balances
	go func() {
		for {
			for _, v := range allWatching {
				v.Balance = GetEthBalance(v.Address).String()
			}
			time.Sleep(15 * time.Second)
		}
	}()

	fmt.Printf("ETHexporter has started on port %v using Geth server: %v\n", port, gethUrl)
	http.HandleFunc("/metrics", MetricsHttp)
	panic(http.ListenAndServe("0.0.0.0:"+port, nil))
}
