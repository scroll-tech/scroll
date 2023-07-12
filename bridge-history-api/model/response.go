package model

import "bridge-history-api/service"

// Data defines the return struct of apis
type Data struct {
	Result []*service.TxHistoryInfo `json:"result"`
	Total  uint64                   `json:"total"`
}

// QueryByAddressResponse defines the query by address api response struct
type QueryByAddressResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}

// QueryByHashResponse defines the query by hash api response struct
type QueryByHashResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}
