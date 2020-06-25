package market

import (
	"github.com/Bundle-App/blockatlas/pkg/blockatlas"
	"github.com/Bundle-App/blockatlas/storage"
)

type Provider interface {
	Init(storage.Market) error
	GetId() string
	GetUpdateTime() string
	GetData() (blockatlas.Tickers, error)
	GetLogType() string
}

type Providers map[int]Provider

func (ps Providers) GetPriority(providerId string) int {
	for priority, provider := range ps {
		if provider.GetId() == providerId {
			return priority
		}
	}
	return -1
}
