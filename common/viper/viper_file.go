package viper

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"scroll-tech/common/viper/internal/encoding"
	"scroll-tech/common/viper/internal/encoding/dotenv"
	"scroll-tech/common/viper/internal/encoding/json"
	"scroll-tech/common/viper/internal/encoding/toml"
	"scroll-tech/common/viper/internal/encoding/yaml"
)

var (
	decoders = map[string]encoding.Decoder{
		"yaml":   yaml.Codec{},
		"yml":    yaml.Codec{},
		"json":   json.Codec{},
		"toml":   toml.Codec{},
		"dotenv": dotenv.Codec{},
		"env":    dotenv.Codec{},
	}
)

// Reset : reset viper config.
func (v *Viper) Reset() {
	if !v.isRoot {
		// TODO: only root node can be reset.
	}
	v.configFile = ""
	v.data = sync.Map{}
	//v.data = cmap.New()
}

// SetConfigType : set config type.
func (v *Viper) SetConfigType(in string) {
	_, ok := decoders[in]
	if !ok {
		return
	}
	v.configType = in
}

// SetConfigFile : set config file.
func (v *Viper) SetConfigFile(in string) {
	if in != "" {
		v.configFile = in
	}
}

// ReadInFile : read config file.
func (v *Viper) ReadInFile() error {
	if !v.isRoot {
		return fmt.Errorf("only root node can call this func")
	}

	config := make(map[string]interface{})
	data, err := os.ReadFile(v.configFile)
	if err != nil {
		return err
	}
	if err = v.unmarshal(bytes.NewReader(data), config); err != nil {
		return err
	}

	v.flush(config)
	return nil
}

// ReadConfig : read config by io reader.
func (v *Viper) ReadConfig(in io.Reader) error {
	config := make(map[string]interface{})
	if err := v.unmarshal(in, config); err != nil {
		return err
	}

	v.flush(config)
	return nil
}

func (v *Viper) unmarshal(in io.Reader, c map[string]interface{}) error {
	buf := bytes.Buffer{}
	if _, err := buf.ReadFrom(in); err != nil {
		return err
	}

	decoder, ok := decoders[getConfigType(v.configFile, v.configType)]
	if !ok {
		return fmt.Errorf("don't support this kind of data")
	}
	return decoder.Decode(buf.Bytes(), c)
}

func (v *Viper) flush(m map[string]interface{}) {
	for key, val := range m {
		switch val.(type) {
		case map[interface{}]interface{}, map[string]interface{}:
			vp := v.Sub(key)
			if vp == nil {
				vp = &Viper{}
				v.data.Store(key, vp)
			}
			mp, ok := val.(map[string]interface{})
			if !ok {
				mp = make(map[string]interface{})
				for k, v := range val.(map[interface{}]interface{}) {
					mp[fmt.Sprintf("%v", k)] = v
				}
			}
			vp.flush(mp)
		default:
			v.data.Store(key, val)
		}
	}
}
