package ethereum

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/Bundle-App/blockatlas/coin"
	"github.com/Bundle-App/blockatlas/pkg/address"
	"github.com/Bundle-App/blockatlas/pkg/blockatlas"
	"github.com/Bundle-App/blockatlas/pkg/logger"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

var (
	supportedTypes = map[string]bool{"ERC721": true, "ERC1155": true}
	slugTokens     = map[string]bool{"ERC1155": true}
)

type Platform struct {
	client            Client
	collectionsClient CollectionsClient
	CoinIndex         uint
	RpcURL            string
}

func (p *Platform) Init() error {
	handle := coin.Coins[p.CoinIndex].Handle

	coinVar := fmt.Sprintf("%s.api", handle)
	p.client = Client{blockatlas.InitClient(viper.GetString(coinVar))}

	collectionsApiVar := fmt.Sprintf("%s.collections_api", handle)
	p.collectionsClient = CollectionsClient{blockatlas.InitClient(viper.GetString(collectionsApiVar))}

	collectionsApiKeyVar := fmt.Sprintf("%s.collections_api_key", handle)
	p.collectionsClient.Headers["X-API-KEY"] = viper.GetString(collectionsApiKeyVar)

	p.RpcURL = viper.GetString("ethereum.rpc")
	return nil
}

func (p *Platform) Coin() coin.Coin {
	return coin.Coins[p.CoinIndex]
}

func (p *Platform) RegisterRoutes(router gin.IRouter) {
	router.GET("/:address", func(c *gin.Context) {
		p.getTransactions(c)
	})
}

func (p *Platform) getTransactions(c *gin.Context) {
	token := c.Query("token")
	address := c.Param("address")
	var srcPage *Page
	var err error

	if token != "" {
		srcPage, err = p.client.GetTxsWithContract(address, token)
	} else {
		srcPage, err = p.client.GetTxs(address)
	}

	if apiError(c, err) {
		return
	}

	var txs []blockatlas.Tx
	for _, srcTx := range srcPage.Docs {
		txs = AppendTxs(txs, &srcTx, p.CoinIndex)
	}

	page := blockatlas.TxPage(txs)
	page.Sort()
	c.JSON(http.StatusOK, &page)
}

func extractBase(srcTx *Doc, coinIndex uint) (base blockatlas.Tx, ok bool) {
	var status blockatlas.Status
	var errReason string
	if srcTx.Error == "" {
		status = blockatlas.StatusCompleted
	} else {
		status = blockatlas.StatusFailed
		errReason = srcTx.Error
	}

	fee := calcFee(srcTx.GasPrice, srcTx.GasUsed)

	base = blockatlas.Tx{
		ID:       srcTx.ID,
		Coin:     coinIndex,
		From:     address.EIP55Checksum(srcTx.From),
		To:       address.EIP55Checksum(srcTx.To),
		Fee:      blockatlas.Amount(fee),
		Date:     srcTx.Timestamp,
		Block:    srcTx.BlockNumber,
		Status:   status,
		Error:    errReason,
		Sequence: srcTx.Nonce,
	}
	return base, true
}

func AppendTxs(in []blockatlas.Tx, srcTx *Doc, coinIndex uint) (out []blockatlas.Tx) {
	out = in
	baseTx, ok := extractBase(srcTx, coinIndex)
	if !ok {
		return
	}

	// Native ETH transaction
	if len(srcTx.Ops) == 0 && srcTx.Input == "0x" {
		transferTx := baseTx
		transferTx.Meta = blockatlas.Transfer{
			Value:    blockatlas.Amount(srcTx.Value),
			Symbol:   coin.Coins[coinIndex].Symbol,
			Decimals: coin.Coins[coinIndex].Decimals,
		}
		out = append(out, transferTx)
	}

	// Smart Contract Call
	if len(srcTx.Ops) == 0 && srcTx.Input != "0x" {
		contractTx := baseTx
		contractTx.Meta = blockatlas.ContractCall{
			Input: srcTx.Input,
			Value: srcTx.Value,
		}
		out = append(out, contractTx)
	}

	if len(srcTx.Ops) == 0 {
		return
	}
	op := &srcTx.Ops[0]

	if op.Type == blockatlas.TxTokenTransfer && op.Contract != nil {
		tokenTx := baseTx

		tokenTx.Meta = blockatlas.TokenTransfer{
			Name:     op.Contract.Name,
			Symbol:   op.Contract.Symbol,
			TokenID:  address.EIP55Checksum(op.Contract.Address),
			Decimals: op.Contract.Decimals,
			Value:    blockatlas.Amount(op.Value),
			From:     op.From,
			To:       op.To,
		}
		out = append(out, tokenTx)
	}
	return
}

func calcFee(gasPrice string, gasUsed string) string {
	var gasPriceBig, gasUsedBig, feeBig big.Int

	gasPriceBig.SetString(gasPrice, 10)
	gasUsedBig.SetString(gasUsed, 10)

	feeBig.Mul(&gasPriceBig, &gasUsedBig)

	return feeBig.String()
}

func apiError(c *gin.Context, err error) bool {
	if err != nil {
		logger.Error(err, "Unhandled error")
		c.AbortWithStatus(http.StatusInternalServerError)
		return true
	}
	return false
}

func (p *Platform) CurrentBlockNumber() (int64, error) {
	return p.client.CurrentBlockNumber()
}

func (p *Platform) GetBlockByNumber(num int64) (*blockatlas.Block, error) {
	if srcPage, err := p.client.GetBlockByNumber(num); err == nil {
		var txs []blockatlas.Tx
		for _, srcTx := range srcPage {
			txs = AppendTxs(txs, &srcTx, p.CoinIndex)
		}
		return &blockatlas.Block{
			Number: num,
			ID:     strconv.FormatInt(num, 10),
			Txs:    txs,
		}, nil
	} else {
		logger.Error(fmt.Sprintf("ATLAS_LOGS_ERROR : (ETH)GetTransactionsByBlock FAILED | BlockNumber: %d | - Error: %v | ",num, err))
		return nil, err
	}
}

func (p *Platform) GetCollections(owner string) (blockatlas.CollectionPage, error) {
	collections, err := p.collectionsClient.GetCollections(owner)
	if err != nil {
		return nil, err
	}
	page := NormalizeCollectionPage(collections, p.CoinIndex, owner)
	return page, nil
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func (p *Platform) OldGetCollections(owner string) (blockatlas.CollectionPage, error) {
	collections, err := p.collectionsClient.GetCollections(owner)
	if err != nil {
		return nil, err
	}
	page := OldNormalizeCollectionPage(collections, p.CoinIndex, owner)
	return page, nil
}

func (p *Platform) GetCollectibles(owner, collectibleID string) (blockatlas.CollectiblePage, error) {
	collection, items, err := p.collectionsClient.GetCollectibles(owner, collectibleID)
	if err != nil {
		return nil, err
	}
	page := NormalizeCollectiblePage(collection, items, p.CoinIndex)
	return page, nil
}

func NormalizeCollectionPage(collections []Collection, coinIndex uint, owner string) (page blockatlas.CollectionPage) {
	for _, collection := range collections {
		if len(collection.Contracts) == 0 {
			continue
		}
		item := NormalizeCollection(collection, coinIndex, owner)
		if _, ok := supportedTypes[item.Type]; !ok {
			continue
		}
		page = append(page, item)
	}
	return
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func (p *Platform) OldGetCollectibles(owner, collectibleID string) (blockatlas.CollectiblePage, error) {
	collection, items, err := p.collectionsClient.OldGetCollectibles(owner, collectibleID)
	if err != nil {
		return nil, err
	}
	page := OldNormalizeCollectiblePage(collection, items, p.CoinIndex)
	return page, nil
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func OldNormalizeCollectionPage(collections []Collection, coinIndex uint, owner string) (page blockatlas.CollectionPage) {
	for _, collection := range collections {
		if len(collection.Contracts) == 0 {
			continue
		}
		item := OldNormalizeCollection(collection, coinIndex, owner)
		if _, ok := supportedTypes[item.Type]; !ok {
			continue
		}
		page = append(page, item)
	}
	return
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func OldNormalizeCollection(c Collection, coinIndex uint, owner string) blockatlas.Collection {
	if len(c.Contracts) == 0 {
		return blockatlas.Collection{}
	}

	description := blockatlas.GetValidParameter(c.Description, c.Contracts[0].Description)
	symbol := blockatlas.GetValidParameter(c.Contracts[0].Symbol, "")
	collectionId := blockatlas.GetValidParameter(c.Contracts[0].Address, "")
	version := blockatlas.GetValidParameter(c.Contracts[0].NftVersion, "")
	collectionType := blockatlas.GetValidParameter(c.Contracts[0].Type, "")
	if _, ok := slugTokens[collectionType]; ok {
		collectionId = createCollectionId(collectionId, c.Slug)
	}

	return blockatlas.Collection{
		Name:            c.Name,
		Symbol:          symbol,
		Slug:            c.Slug,
		ImageUrl:        c.ImageUrl,
		Description:     description,
		ExternalLink:    c.ExternalUrl,
		Total:           int(c.Total.Int64()),
		Id:              collectionId,
		CategoryAddress: collectionId,
		Address:         owner,
		Version:         version,
		Coin:            coinIndex,
		Type:            collectionType,
	}
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func OldNormalizeCollectible(c *Collection, a Collectible, coinIndex uint) blockatlas.Collectible {
	// TODO: fix unprotected code
	address := blockatlas.GetValidParameter(c.Contracts[0].Address, "")
	collectionType := blockatlas.GetValidParameter(c.Contracts[0].Type, "")
	collectionID := address
	if _, ok := slugTokens[collectionType]; ok {
		collectionID = createCollectionId(address, c.Slug)
	}
	externalLink := blockatlas.GetValidParameter(a.ExternalLink, a.AssetContract.ExternalLink)
	id := strings.Join([]string{a.AssetContract.Address, a.TokenId}, "-")
	return blockatlas.Collectible{
		ID:               id,
		CollectionID:     collectionID,
		ContractAddress:  address,
		TokenID:          a.TokenId,
		CategoryContract: a.AssetContract.Address,
		Name:             a.Name,
		Category:         c.Name,
		ImageUrl:         a.ImagePreviewUrl,
		ProviderLink:     a.Permalink,
		ExternalLink:     externalLink,
		Type:             collectionType,
		Description:      a.Description,
		Coin:             coinIndex,
	}
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func createCollectionId(address, slug string) string {
	return fmt.Sprintf("%s---%s", address, slug)
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func getCollectionId(collectionId string) string {
	s := strings.Split(collectionId, "---")
	if len(s) != 2 {
		return collectionId
	}
	return s[1]
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func OldNormalizeCollectiblePage(c *Collection, srcPage []Collectible, coinIndex uint) (page blockatlas.CollectiblePage) {
	if len(c.Contracts) == 0 {
		return
	}
	for _, src := range srcPage {
		item := OldNormalizeCollectible(c, src, coinIndex)
		if _, ok := supportedTypes[item.Type]; !ok {
			continue
		}
		page = append(page, item)
	}
	return
}

func NormalizeCollection(c Collection, coinIndex uint, owner string) blockatlas.Collection {
	if len(c.Contracts) == 0 {
		return blockatlas.Collection{}
	}

	description := blockatlas.GetValidParameter(c.Description, c.Contracts[0].Description)
	symbol := blockatlas.GetValidParameter(c.Contracts[0].Symbol, "")
	version := blockatlas.GetValidParameter(c.Contracts[0].NftVersion, "")
	collectionType := blockatlas.GetValidParameter(c.Contracts[0].Type, "")

	return blockatlas.Collection{
		Name:            c.Name,
		Symbol:          symbol,
		Slug:            c.Slug,
		ImageUrl:        c.ImageUrl,
		Description:     description,
		ExternalLink:    c.ExternalUrl,
		Total:           int(c.Total.Int64()),
		Id:              c.Slug,
		CategoryAddress: c.Slug,
		Address:         owner,
		Version:         version,
		Coin:            coinIndex,
		Type:            collectionType,
	}
}

func NormalizeCollectiblePage(c *Collection, srcPage []Collectible, coinIndex uint) (page blockatlas.CollectiblePage) {
	if len(c.Contracts) == 0 {
		return
	}
	for _, src := range srcPage {
		item := NormalizeCollectible(c, src, coinIndex)
		if _, ok := supportedTypes[item.Type]; !ok {
			continue
		}
		page = append(page, item)
	}
	return
}

func NormalizeCollectible(c *Collection, a Collectible, coinIndex uint) blockatlas.Collectible {
	// TODO: fix unprotected code
	address := blockatlas.GetValidParameter(c.Contracts[0].Address, "")
	collectionType := blockatlas.GetValidParameter(c.Contracts[0].Type, "")
	externalLink := blockatlas.GetValidParameter(a.ExternalLink, a.AssetContract.ExternalLink)
	id := strings.Join([]string{a.AssetContract.Address, a.TokenId}, "-")
	return blockatlas.Collectible{
		ID:               id,
		CollectionID:     c.Slug,
		ContractAddress:  address,
		TokenID:          a.TokenId,
		CategoryContract: a.AssetContract.Address,
		Name:             a.Name,
		Category:         c.Name,
		ImageUrl:         a.ImagePreviewUrl,
		ProviderLink:     a.Permalink,
		ExternalLink:     externalLink,
		Type:             collectionType,
		Description:      a.Description,
		Coin:             coinIndex,
	}
}

func (p *Platform) GetTokenListByAddress(address string) (blockatlas.TokenPage, error) {
	account, err := p.client.GetTokens(address)
	if err != nil {
		return nil, err
	}
	return NormalizeTokens(account.Docs, *p), nil
}

// NormalizeToken converts a Ethereum token into the generic model
func NormalizeToken(srcToken *Token, coinIndex uint) (t blockatlas.Token, ok bool) {
	t = blockatlas.Token{
		Name:     srcToken.Contract.Name,
		Symbol:   srcToken.Contract.Symbol,
		TokenID:  srcToken.Contract.Contract,
		Coin:     coinIndex,
		Decimals: srcToken.Contract.Decimals,
		Type:     blockatlas.TokenTypeERC20,
	}

	return t, true
}

// NormalizeTxs converts multiple Ethereum tokens
func NormalizeTokens(srcTokens []Token, p Platform) []blockatlas.Token {
	tokenPage := make([]blockatlas.Token, 0)
	for _, srcToken := range srcTokens {
		token, ok := NormalizeToken(&srcToken, p.CoinIndex)
		if !ok {
			continue
		}
		tokenPage = append(tokenPage, token)
	}
	return tokenPage
}
