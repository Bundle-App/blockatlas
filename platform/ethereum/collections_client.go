package ethereum

import (
	"github.com/Bundle-App/blockatlas/pkg/blockatlas"
	"github.com/Bundle-App/blockatlas/pkg/errors"
	"net/url"
	"strconv"
	"strings"
)

type CollectionsClient struct {
	blockatlas.Request
}

func (c CollectionsClient) GetCollections(owner string) (page []Collection, err error) {
	query := url.Values{
		"asset_owner": {owner},
		"limit":       {"1000"},
	}
	err = c.Get(&page, "api/v1/collections", query)
	return
}

func (c CollectionsClient) GetCollectibles(owner string, collectibleID string) (*Collection, []Collectible, error) {
	collections, err := c.GetCollections(owner)
	if err != nil {
		return nil, nil, err
	}
	collection := searchCollection(collections, collectibleID)
	if collection == nil {
		return nil, nil, errors.E("collectible not found", errors.TypePlatformClient,
			errors.Params{"collectibleID": collectibleID}).PushToSentry()
	}

	query := url.Values{
		"owner": {owner},
		"limit": {strconv.Itoa(300)},
	}

	query.Set("collection", collection.Slug)

	var page CollectiblePage
	err = c.Get(&page, "api/v1/assets", query)
	return collection, page.Collectibles, err
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func (c CollectionsClient) OldGetCollectibles(owner string, collectibleID string) (*Collection, []Collectible, error) {
	collections, err := c.GetCollections(owner)
	if err != nil {
		return nil, nil, err
	}
	id := getCollectionId(collectibleID)
	collection := oldSearchCollection(collections, id)
	if collection == nil {
		return nil, nil, errors.E("collectible not found", errors.TypePlatformClient,
			errors.Params{"collectibleID": collectibleID}).PushToSentry()
	}

	query := url.Values{
		"owner": {owner},
		"limit": {strconv.Itoa(300)},
	}

	for _, i := range collection.Contracts {
		if _, ok := slugTokens[i.Type]; ok {
			query.Set("collection", collection.Slug)
			break
		}
		query.Add("asset_contract_addresses", i.Address)
	}

	var page CollectiblePage
	err = c.Get(&page, "api/v1/assets", query)
	return collection, page.Collectibles, err
}

func searchCollection(collections []Collection, collectibleID string) *Collection {
	for _, i := range collections {
		if strings.EqualFold(i.Slug, collectibleID) {
			return &i
		}
	}
	return nil
}

//TODO: remove once most of the clients will be updated (deadline: March 17th)
func oldSearchCollection(collections []Collection, collectibleID string) *Collection {
	for _, i := range collections {
		if strings.EqualFold(i.Slug, collectibleID) {
			return &i
		}
		for _, contract := range i.Contracts {
			if strings.EqualFold(contract.Address, collectibleID) {
				return &i
			}
		}
	}
	return nil
}
