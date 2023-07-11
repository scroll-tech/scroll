package model

import "bridge-history-api/service"

type Data struct {
	Result []*service.TxHistoryInfo `json:"result"`
	Total  uint64                   `json:"total"`
}

type QueryByAddressResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}

type QueryByHashResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}
