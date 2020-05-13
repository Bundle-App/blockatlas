package platform

import (
	"fmt"
	"github.com/Bundle-App/blockatlas/platform/cosmos"

	"github.com/spf13/viper"
	"github.com/Bundle-App/blockatlas/pkg/blockatlas"
	"github.com/Bundle-App/blockatlas/pkg/logger"

	"github.com/Bundle-App/blockatlas/coin"
	"github.com/Bundle-App/blockatlas/platform/aeternity"
	"github.com/Bundle-App/blockatlas/platform/aion"
	"github.com/Bundle-App/blockatlas/platform/algorand"
	"github.com/Bundle-App/blockatlas/platform/binance"
	"github.com/Bundle-App/blockatlas/platform/bitcoin"
	"github.com/Bundle-App/blockatlas/platform/ethereum"
	"github.com/Bundle-App/blockatlas/platform/fio"
	"github.com/Bundle-App/blockatlas/platform/harmony"
	"github.com/Bundle-App/blockatlas/platform/icon"
	"github.com/Bundle-App/blockatlas/platform/iotex"
	"github.com/Bundle-App/blockatlas/platform/nano"
	"github.com/Bundle-App/blockatlas/platform/nebulas"
	"github.com/Bundle-App/blockatlas/platform/nimiq"
	"github.com/Bundle-App/blockatlas/platform/ontology"
	"github.com/Bundle-App/blockatlas/platform/ripple"
	"github.com/Bundle-App/blockatlas/platform/stellar"
	"github.com/Bundle-App/blockatlas/platform/tezos"
	"github.com/Bundle-App/blockatlas/platform/theta"
	"github.com/Bundle-App/blockatlas/platform/tron"
	"github.com/Bundle-App/blockatlas/platform/vechain"
	"github.com/Bundle-App/blockatlas/platform/waves"
	"github.com/Bundle-App/blockatlas/platform/zilliqa"
)

var platformList = []blockatlas.Platform{
	&binance.Platform{},
	&nimiq.Platform{},
	&ripple.Platform{},
	&stellar.Platform{CoinIndex: coin.XLM},
	&stellar.Platform{CoinIndex: coin.KIN},
	&ethereum.Platform{CoinIndex: coin.ETH},
	&ethereum.Platform{CoinIndex: coin.ETC},
	&ethereum.Platform{CoinIndex: coin.POA},
	&ethereum.Platform{CoinIndex: coin.CLO},
	&ethereum.Platform{CoinIndex: coin.GO},
	&ethereum.Platform{CoinIndex: coin.WAN},
	&ethereum.Platform{CoinIndex: coin.TOMO},
	&ethereum.Platform{CoinIndex: coin.TT},
	&cosmos.Platform{CoinIndex: coin.ATOM},
	&cosmos.Platform{CoinIndex: coin.KAVA},
	&tezos.Platform{},
	&aion.Platform{},
	&icon.Platform{},
	&iotex.Platform{},
	&ontology.Platform{},
	&theta.Platform{},
	&tron.Platform{},
	&vechain.Platform{},
	&zilliqa.Platform{},
	&waves.Platform{},
	&aeternity.Platform{},
	&bitcoin.Platform{CoinIndex: coin.BTC},
	&bitcoin.Platform{CoinIndex: coin.LTC},
	&bitcoin.Platform{CoinIndex: coin.BCH},
	&bitcoin.Platform{CoinIndex: coin.DASH},
	&bitcoin.Platform{CoinIndex: coin.DOGE},
	&bitcoin.Platform{CoinIndex: coin.ZEC},
	&bitcoin.Platform{CoinIndex: coin.XZC},
	&bitcoin.Platform{CoinIndex: coin.VIA},
	&bitcoin.Platform{CoinIndex: coin.RVN},
	&bitcoin.Platform{CoinIndex: coin.QTUM},
	&bitcoin.Platform{CoinIndex: coin.GRS},
	&bitcoin.Platform{CoinIndex: coin.ZEL},
	&bitcoin.Platform{CoinIndex: coin.DCR},
	&bitcoin.Platform{CoinIndex: coin.DGB},
	&nebulas.Platform{},
	&fio.Platform{},
	&algorand.Platform{},
	&nano.Platform{},
	&harmony.Platform{},
	//&polkadot.Platform{CoinIndex: coin.KSM},
}

// Platforms contains all registered platforms by handle
var Platforms map[string]blockatlas.Platform

// BlockAPIs contain platforms with block services
var BlockAPIs map[string]blockatlas.BlockAPI

// StakeAPIs contain platforms with staking services
var StakeAPIs map[string]blockatlas.StakeAPI

// CustomAPIs contain platforms with custom HTTP services
var CustomAPIs map[string]blockatlas.CustomAPI

// NamingAPIs contain platforms which support naming services
var NamingAPIs map[uint64]blockatlas.NamingServiceAPI

// CollectionAPIs contain platforms which collections services
var CollectionAPIs map[uint]blockatlas.CollectionAPI

func Init() {
	Platforms = make(map[string]blockatlas.Platform)
	BlockAPIs = make(map[string]blockatlas.BlockAPI)
	StakeAPIs = make(map[string]blockatlas.StakeAPI)
	CustomAPIs = make(map[string]blockatlas.CustomAPI)
	NamingAPIs = make(map[uint64]blockatlas.NamingServiceAPI)
	CollectionAPIs = make(map[uint]blockatlas.CollectionAPI)

	for _, platform := range platformList {
		handle := platform.Coin().Handle
		apiURL := fmt.Sprintf("%s.api", handle)

		if !viper.IsSet(apiURL) {
			continue
		}
		if viper.GetString(apiURL) == "" {
			continue
		}

		p := logger.Params{
			"platform": handle,
			"coin":     platform.Coin(),
		}

		if _, exists := Platforms[handle]; exists {
			logger.Fatal("Duplicate handle", p)
		}

		err := platform.Init()
		if err != nil {
			logger.Error("Failed to initialize API", err, p)
		}

		Platforms[handle] = platform

		if blockAPI, ok := platform.(blockatlas.BlockAPI); ok {
			BlockAPIs[handle] = blockAPI
		}
		if stakeAPI, ok := platform.(blockatlas.StakeAPI); ok {
			StakeAPIs[handle] = stakeAPI
		}
		if customAPI, ok := platform.(blockatlas.CustomAPI); ok {
			CustomAPIs[handle] = customAPI
		}
		if namingAPI, ok := platform.(blockatlas.NamingServiceAPI); ok {
			NamingAPIs[uint64(platform.Coin().ID)] = namingAPI
		}
		if collectionAPI, ok := platform.(blockatlas.CollectionAPI); ok {
			CollectionAPIs[platform.Coin().ID] = collectionAPI
		}
	}
}
