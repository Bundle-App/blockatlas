package observer

import (
	"encoding/json"
	"fmt"
	"github.com/Bundle-App/blockatlas/coin"
	"github.com/Bundle-App/blockatlas/pkg/blockatlas"
	"github.com/Bundle-App/blockatlas/platform/bitcoin"
	"io/ioutil"
	"testing"
)

func GetTxsTT(block *blockatlas.Block) map[string]*blockatlas.TxSet {
	txMap := make(map[string]*blockatlas.TxSet)
	for i := 0; i < len(block.Txs); i++ {
		addresses := block.Txs[i].GetAddresses()
		addresses = append(addresses, block.Txs[i].GetUtxoAddresses()...)
		for _, address := range addresses {
			if txMap[address] == nil {
				txMap[address] = new(blockatlas.TxSet)
			}
			txMap[address].Add(&block.Txs[i])
		}
	}
	return txMap
}

func ProcessBlockTT(block *blockatlas.Block, subs []blockatlas.Subscription) {
	txMap := GetTxsTT(block)
	if len(txMap) == 0 {
		return
	}

	// Build list of unique addresses
	var addresses []string
	for address := range txMap {
		if len(address) == 0 {
			continue
		}
		addresses = append(addresses, address)
	}

	for _, sub := range subs {
		tx, ok := txMap[sub.Address]
		if !ok {
			continue
		}
		for _, tx := range tx.Txs() {
			tx.Direction = getDirection(tx, sub.Address)
			inferUtxoValue(&tx, sub.Address, coin.BTC)
			fmt.Printf("\nAddr: %s => Tx: %v\n", sub.Address, tx)
		}
	}
}


var subs = []blockatlas.Subscription{
	{
		Coin:    coin.BTC,
		Address: "1JhGgCsK2DwHMhmhA2gVnPo6KwNtuuEhph",
		Webhook: "",
	},
	{
		Coin:    coin.BTC,
		Address: "16FxrKZq8sceayWt3bemEjwRRugQu6iK82",
		Webhook: "",
	},

}


func TestProcessBlock(t *testing.T) {
	var data []byte
	data, err := ioutil.ReadFile("test_block_txs.json")
	if err != nil {
		fmt.Printf("Error loading file: %v", err)
	}

	var  block bitcoin.TransactionsList
	err = json.Unmarshal(data, &block)
	if err != nil {
		fmt.Printf("Error parsing: %v", err)
	}

	var normalized []blockatlas.Tx
	for _, tx := range block.TransactionList() {
		normalized = append(normalized, bitcoin.NormalizeTransaction(tx, 0))
	}

	blk := blockatlas.Block{
		ID:     block.Hash,
		Txs:    normalized,
	}

	ProcessBlockTT(&blk, subs)
}
