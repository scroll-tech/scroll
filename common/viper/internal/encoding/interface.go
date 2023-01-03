package encoding

// Encoder encodes the contents of v into a byte representation.
// It's primarily used for encoding a map[string]interface{} into a file format.
type Encoder interface {
	Encode(v map[string]interface{}) ([]byte, error)
}

// Decoder decodes the contents of b into v.
// It's primarily used for decoding contents of a file into a map[string]interface{}.
type Decoder interface {
	Decode(b []byte, v map[string]interface{}) error
}
