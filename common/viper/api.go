package viper

import "fmt"

func (v *Viper) Get(key string) interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fmt.Println("Get key: ", key, " value: ", v.Viper.Get(key))
	return v.Viper.Get(key)
}

func (v *Viper) GetBool(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fmt.Println("GetBool key: ", key, " value: ", v.Viper.GetBool(key))
	return v.Viper.GetBool(key)
}

func (v *Viper) GetFloat64(key string) float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fmt.Println("GetFloat64 key: ", key, " value: ", v.Viper.GetFloat64(key))
	return v.Viper.GetFloat64(key)
}

func (v *Viper) GetInt(key string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fmt.Println("GetInt key: ", key, " value: ", v.Viper.GetInt(key))
	return v.Viper.GetInt(key)
}

func (v *Viper) GetIntSlice(key string) []int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fmt.Println("GetIntSlice key: ", key, " value: ", v.Viper.GetIntSlice(key))
	return v.Viper.GetIntSlice(key)
}

func (v *Viper) GetString(key string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fmt.Println("GetString key: ", key, " value: ", v.Viper.GetString(key))
	return v.Viper.GetString(key)
}

func (v *Viper) GetStringSlice(key string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fmt.Println("GetStringSlice key: ", key, " value: ", v.Viper.GetStringSlice(key))
	return v.Viper.GetStringSlice(key)
}

//
/*func (v *Viper) GetBigInt(key string) *big.Int {
	val := v.Viper.Get(key)
	switch val.(type) {
	case int:
	case string:

	}
}*/
