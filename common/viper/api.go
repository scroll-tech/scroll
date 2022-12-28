package viper

func (v *Viper) Get(key string) interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Viper.Get(key)
}

func (v *Viper) GetBool(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Viper.GetBool(key)
}

func (v *Viper) GetFloat64(key string) float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Viper.GetFloat64(key)
}

func (v *Viper) GetInt(key string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Viper.GetInt(key)
}

func (v *Viper) GetIntSlice(key string) []int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Viper.GetIntSlice(key)
}

func (v *Viper) GetString(key string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Viper.GetString(key)
}

func (v *Viper) GetStringSlice(key string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
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
