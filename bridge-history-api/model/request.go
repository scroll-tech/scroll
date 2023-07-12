package model

// QueryByAddressRequest defines the query by address api request struct
type QueryByAddressRequest struct {
	Address string `url:"address"`
	Offset  int    `url:"offset"`
	Limit   int    `url:"limit"`
}

// QueryByHashRequest defines the query by hash api request sturct
type QueryByHashRequest struct {
	Txs []string `url:"txs"`
}
