package viper

func (v *Viper) Get(key string) interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("Get key: ", key, " value: ", v.vp.Get(key))
	return v.vp.Get(key)
}

func (v *Viper) GetBool(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetBool key: ", key, " value: ", v.vp.GetBool(key))
	return v.vp.GetBool(key)
}

func (v *Viper) GetFloat64(key string) float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetFloat64 key: ", key, " value: ", v.vp.GetFloat64(key))
	return v.vp.GetFloat64(key)
}

func (v *Viper) GetInt(key string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetInt key: ", key, " value: ", v.vp.GetInt(key))
	return v.vp.GetInt(key)
}

func (v *Viper) GetInt64(key string) int64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetInt key: ", key, " value: ", v.vp.GetInt64(key))
	return v.vp.GetInt64(key)
}

func (v *Viper) GetIntSlice(key string) []int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetIntSlice key: ", key, " value: ", v.vp.GetIntSlice(key))
	return v.vp.GetIntSlice(key)
}

func (v *Viper) GetString(key string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetString key: ", key, " value: ", v.vp.GetString(key))
	return v.vp.GetString(key)
}

func (v *Viper) GetStringSlice(key string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	//fmt.Println("GetStringSlice key: ", key, " value: ", v.vp.GetStringSlice(key))
	return v.vp.GetStringSlice(key)
}

//
/*func (v *Viper) GetBigInt(key string) *big.Int {
	val := v.vp.Get(key)
	switch val.(type) {
	case int:
	case string:

	}
}*/
