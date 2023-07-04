package controller

import (
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

type QueryClaimableController struct {
	Service service.HistoryService
}

func (c *QueryClaimableController) Get(req model.QueryByAddressRequest) (*model.QueryByAddressResponse, error) {
	message, total, err := c.Service.GetClaimableTxsByAddress(common.HexToAddress(req.Address), int64(req.Offset), int64(req.Limit))
	if err != nil {
		return &model.QueryByAddressResponse{Message: "500", Data: &model.Data{}}, err
	}

	return &model.QueryByAddressResponse{Message: "ok",
		Data: &model.Data{
			Result: message,
			Total:  total,
		}}, nil
}

func (c *QueryAddressController) Get(req model.QueryByAddressRequest) (*model.QueryByAddressResponse, error) {
	message, total, err := c.Service.GetTxsByAddress(common.HexToAddress(req.Address), int64(req.Offset), int64(req.Limit))
	if err != nil {
		return &model.QueryByAddressResponse{Message: "500", Data: &model.Data{}}, err
	}

	return &model.QueryByAddressResponse{Message: "ok",
		Data: &model.Data{
			Result: message,
			Total:  total,
		}}, nil
}

func (c *QueryHashController) Post(req model.QueryByHashRequest) (*model.QueryByHashResponse, error) {
	result, err := c.Service.GetTxsByHashes(req.Txs)
	if err != nil {
		return &model.QueryByHashResponse{Message: "500", Data: &model.Data{}}, err
	}
	return &model.QueryByHashResponse{Message: "ok", Data: &model.Data{Result: result}}, nil
}
