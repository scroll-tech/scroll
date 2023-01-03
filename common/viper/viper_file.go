package viper

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	config "scroll-tech/common/apollo"

	"scroll-tech/common/viper/internal/encoding"
	"scroll-tech/common/viper/internal/encoding/dotenv"
	"scroll-tech/common/viper/internal/encoding/json"
	"scroll-tech/common/viper/internal/encoding/toml"
	"scroll-tech/common/viper/internal/encoding/yaml"
)

var (
	encoders = map[string]encoding.Encoder{
		"yaml":   yaml.Codec{},
		"yml":    yaml.Codec{},
		"json":   json.Codec{},
		"toml":   toml.Codec{},
		"dotenv": dotenv.Codec{},
		"env":    dotenv.Codec{},
	}
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
	v.configType, v.configFile = "", ""
	v.data = sync.Map{}
	//v.data = cmap.New()
}

// SetConfigType : set config type.
func (v *Viper) SetConfigType(tp string) {
	_, ok := decoders[tp]
	if !ok {
		log.Warn("SetConfigType fail, type decoder not found", "type", tp)
		return
	}
	v.configType = tp
}

// SetConfigFile : set config file.
func (v *Viper) SetConfigFile(in string) {
	tp := getConfigType(in)
	_, ok := decoders[tp]
	if !ok {
		log.Warn("SetConfigFile fail, type decoder not found", "type", tp)
		return
	}
	v.configType = tp
	v.configFile = in
}

// ReadInFile : read config file.
func (v *Viper) ReadInFile() error {
	if !v.isRoot {
		return fmt.Errorf("only root node can call this func")
	}

	data, err := os.ReadFile(v.configFile)
	if err != nil {
		return err
	}
	config, err := v.unmarshal(bytes.NewReader(data))
	if err != nil {
		return err
	}

	v.flush(config)
	return nil
}

// ReadConfig : read config by io reader.
func (v *Viper) ReadConfig(in io.Reader) error {
	config, err := v.unmarshal(in)
	if err != nil {
		return err
	}

	v.flush(config)
	return nil
}

// WriteConfigAs : writes current configuration to a given filename.
func (v *Viper) WriteConfigAs(filename string) error {
	if !v.isRoot {
		return fmt.Errorf("sub viper don't support write to file")
	}

	buf := bytes.Buffer{}
	if err := v.marshal(&buf, v.configType); err != nil {
		return err
	}
	return os.WriteFile(filename, buf.Bytes(), 0600|0044)
}

// WriteConfig : marshal and write config to io writer.
func (v *Viper) WriteConfig(out io.Writer) error {
	return v.marshal(out, v.configType)
}

func (v *Viper) unmarshal(in io.Reader) (map[string]interface{}, error) {
	buf := bytes.Buffer{}
	if _, err := buf.ReadFrom(in); err != nil {
		return nil, err
	}

	var decoder encoding.Decoder
	if v.configType != "" {
		decoder = decoders[v.configType]
	} else {
		var ok bool
		decoder, ok = decoders[getConfigType(v.configFile)]
		if !ok {
			return nil, fmt.Errorf("don't support this kind of data")
		}
	}

	c := make(map[string]interface{})
	return c, decoder.Decode(buf.Bytes(), c)
}

func (v *Viper) marshal(out io.Writer, configType string) error {
	c := v.export()
	encoder := encoders[configType]
	data, err := encoder.Encode(c)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

func syncApolloRemoteConfig(remoteCfg string, vp *Viper) {
	agolloClient := config.MustInitApollo()

	for {
		cfgStr := agolloClient.GetStringValue(remoteCfg, "")
		if err := vp.ReadConfig(bytes.NewReader([]byte(cfgStr))); err != nil || cfgStr == "" {
			log.Error("ReadConfig fail", "config", cfgStr, "err", err)
			<-time.After(time.Second * 3)
			continue
		}
		<-time.After(time.Second * 3)
	}
}
