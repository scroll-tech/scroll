package model

// QueryByAddressRequest the request parameter of address api
type QueryByAddressRequest struct {
	Address string `url:"address"`
	Offset  int    `url:"offset"`
	Limit   int    `url:"limit"`
}

// QueryByHashRequest the request parameter of hash api
type QueryByHashRequest struct {
	Txs []string `url:"txs"`
}
