package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"scroll-tech/common/types"
)

func TestResponseDecodeData_GetTaskSchema(t *testing.T) {
	// Arrange: build a dummy payload and wrap it in Response
	in := GetTaskSchema{
		UUID:         "uuid-123",
		TaskID:       "task-abc",
		TaskType:     1,
		UseSnark:     true,
		TaskData:     "dummy-data",
		HardForkName: "cancun",
	}

	resp := types.Response{
		ErrCode: 0,
		ErrMsg:  "",
		Data:    in,
	}

	// Act: JSON round-trip the Response to simulate real HTTP encoding/decoding
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var decoded types.Response
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	var out GetTaskSchema
	if err := decoded.DecodeData(&out); err != nil {
		t.Fatalf("DecodeData error: %v", err)
	}

	// Assert: structs match after decode
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("decoded struct mismatch:\nwant: %+v\n got: %+v", in, out)
	}
}
