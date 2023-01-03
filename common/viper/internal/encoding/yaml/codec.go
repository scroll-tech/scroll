package yaml

// import "gopkg.in/yaml.v2"

// Codec implements the encoding.Encoder and encoding.Decoder interfaces for YAML encoding.
type Codec struct{}

// Encode : marshal yaml.
func (Codec) Encode(v map[string]interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

// Decode : unmarshal yaml.
func (Codec) Decode(b []byte, v map[string]interface{}) error {
	return yaml.Unmarshal(b, &v)
}
