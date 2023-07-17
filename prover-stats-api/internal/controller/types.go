package controller

type Resp struct {
	Code   int    `json:"code"`
	Object []byte `json:"object"`
	Error  error  `json:"error"`
}

const (
	OK  = 1000
	ERR = 1001
)

func Ok(obj []byte) *Resp {
	return &Resp{
		Code:   OK,
		Object: obj,
	}
}

func Err(err error) *Resp {
	return &Resp{
		Code:  ERR,
		Error: err,
	}
}
