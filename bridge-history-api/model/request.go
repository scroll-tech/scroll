package model

type QueryByAddressRequest struct {
	Address string `url:"address"`
	Offset  int    `url:"offset"`
	Limit   int    `url:"limit"`
}

type NFTQueryByAddressRequest struct {
	TokenType string `url:tokenType`
	Address   string `url:"address"`
	Offset    int    `url:"offset"`
	Limit     int    `url:"limit"`
}

type QueryByHashRequest struct {
	Txs []string `url:"txs"`
}
