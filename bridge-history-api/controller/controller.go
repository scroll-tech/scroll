package controller

import (
	"bridge-history-api/db/orm"
	"bridge-history-api/model"
	"bridge-history-api/service"

	"github.com/ethereum/go-ethereum/common"
)

type QueryAddressController struct {
	Service service.HistoryService
}

type QueryHashController struct {
	Service service.HistoryService
}

type QueryNFTAddressController struct {
	Service service.HistoryService
}

func (c *QueryAddressController) Get(req model.QueryByAddressRequest) (*model.QueryByAddressResponse, error) {
	message, err := c.Service.GetERC20TxsByAddress(common.HexToAddress(req.Address), int64(req.Offset), int64(req.Limit))
	if err != nil {
		return &model.QueryByAddressResponse{Message: "500", Data: &model.Data{}}, err
	}

	return &model.QueryByAddressResponse{Message: "ok",
		Data: &model.Data{
			Result: message.Results,
			Total:  len(message.Results),
		}}, nil
}

func (c *QueryNFTAddressController) Get(req model.NFTQueryByAddressRequest) (*model.QueryByAddressResponse, error) {
	asset := orm.NewAssetType(req.TokenType)
	message, err := c.Service.GetNFTTxsByAddress(common.HexToAddress(req.Address), int64(req.Offset), int64(req.Limit), asset)
	if err != nil {
		return &model.QueryByAddressResponse{Message: "500", Data: &model.Data{}}, err
	}

	return &model.QueryByAddressResponse{Message: "ok",
		Data: &model.Data{
			Result: message.Results,
			Total:  len(message.Results),
		}}, nil
}

func (c *QueryHashController) Post(req model.QueryByHashRequest) (*model.QueryByHashResponse, error) {
	result, err := c.Service.GetTxsByHashes(req.Txs)
	if err != nil {
		return &model.QueryByHashResponse{Message: "500", Data: &model.Data{}}, err
	}
	return &model.QueryByHashResponse{Message: "ok", Data: &model.Data{Result: result.Results}}, nil
}
