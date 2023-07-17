package controller

type Resp struct {
	Code   int `json:"code"`
	Object any `json:"object"`
}

const (
	OK  = 1000
	ERR = 1001
)

func Ok(obj any) *Resp {
	return &Resp{
		Code:   OK,
		Object: obj,
	}
}

func Err(obj any) *Resp {
	return &Resp{
		Code:   ERR,
		Object: obj,
	}
}
