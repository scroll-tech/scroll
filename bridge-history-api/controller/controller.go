package controller

import (
	"bridge-history-api/model"
	"bridge-history-api/service"

	"github.com/ethereum/go-ethereum/common"
)

// QueryAddressController contains the query by address service
type QueryAddressController struct {
	Service service.HistoryService
}

// QueryHashController contains the query by hash service
type QueryHashController struct {
	Service service.HistoryService
}

// QueryClaimableController contains the query claimable txs service
type QueryClaimableController struct {
	Service service.HistoryService
}

// Get defines the http get method behavior for QueryClaimableController
func (c *QueryClaimableController) Get(req model.QueryByAddressRequest) (*model.QueryByAddressResponse, error) {
	txs, total, err := c.Service.GetClaimableTxsByAddress(common.HexToAddress(req.Address), req.Offset, req.Limit)
	if err != nil {
		return &model.QueryByAddressResponse{Message: "500", Data: &model.Data{}}, err
	}

	return &model.QueryByAddressResponse{Message: "ok",
		Data: &model.Data{
			Result: txs,
			Total:  total,
		}}, nil
}

// Get defines the http get method behavior for QueryAddressController
func (c *QueryAddressController) Get(req model.QueryByAddressRequest) (*model.QueryByAddressResponse, error) {
	message, total, err := c.Service.GetTxsByAddress(common.HexToAddress(req.Address), req.Offset, req.Limit)
	if err != nil {
		return &model.QueryByAddressResponse{Message: "500", Data: &model.Data{}}, err
	}

	return &model.QueryByAddressResponse{Message: "ok",
		Data: &model.Data{
			Result: message,
			Total:  total,
		}}, nil
}

// Post defines the http post method behavior for QueryHashController
func (c *QueryHashController) Post(req model.QueryByHashRequest) (*model.QueryByHashResponse, error) {
	result, err := c.Service.GetTxsByHashes(req.Txs)
	if err != nil {
		return &model.QueryByHashResponse{Message: "500", Data: &model.Data{}}, err
	}
	return &model.QueryByHashResponse{Message: "ok", Data: &model.Data{Result: result}}, nil
}
