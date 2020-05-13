package zilliqa

import (
	CoinType "github.com/Bundle-App/blockatlas/coin"
	"github.com/Bundle-App/blockatlas/pkg/blockatlas"
)

type ZNSResponse struct {
	Addresses map[string]string
}

func (p *Platform) Lookup(coins []uint64, name string) ([]blockatlas.Resolved, error) {
	var result []blockatlas.Resolved
	resp, err := p.udClient.LookupName(name)
	if err != nil {
		return result, err
	}
	for _, coin := range coins {
		symbol := CoinType.Coins[uint(coin)].Symbol
		result = append(result, blockatlas.Resolved{Coin: coin, Result: resp.Addresses[symbol]})
	}
	return result, nil
}
