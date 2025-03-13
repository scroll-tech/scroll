package message

import (
	"fmt"
	"testing"
)

func TestBytes48(t *testing.T) {
	ti := &Byte48{}
	ti.UnmarshalText([]byte("0x1"))
	if s, err := ti.MarshalText(); err == nil {
		if len(s) != 98 {
			panic(fmt.Sprintf("wrong str: %s", s))
		}
	}
	ti.UnmarshalText([]byte("0x0"))
	if s, err := ti.MarshalText(); err == nil {
		if len(s) != 98 {
			panic(fmt.Sprintf("wrong str: %s", s))
		}
	}
}
