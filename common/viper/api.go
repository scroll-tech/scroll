package viper

func (v *Viper) GetInt(key string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Viper.GetInt(key)
}

//
/*func (v *Viper) GetBigInt(key string) *big.Int {
	val := v.Viper.Get(key)
	switch val.(type) {
	case int:
	case string:

	}
}*/
