package model

type Data struct {
	Result []interface{} `json:"result"`
	Total  int           `json:"total"`
}

type QueryByAddressResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}

type QueryByHashResponse struct {
	Message string `json:"message"`
	Data    *Data  `json:"data"`
}
