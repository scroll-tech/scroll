package model

import "bridge-history-api/service"

// Data the return struct of apis
type Data struct {
	Result []*service.TxHistoryInfo `json:"result"`
	Total  uint64                   `json:"total"`
}

// QueryByAddressResponse the schema of address api response
type QueryByAddressResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}

// QueryByHashResponse the schema of hash api response
type QueryByHashResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}
