package main

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"scroll-tech/common/database"
	"scroll-tech/common/version"
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/cron"
	"scroll-tech/coordinator/internal/route"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"
)

// GetGormDBClient returns a gorm.DB by connecting to the running postgres container
func GetGormDBClient() (*gorm.DB, error) {
	// endpoint, err := t.GetDBEndPoint()
	// if err != nil {
	// 	return nil, err
	// }
	endpoint := "postgres://lmr:@localhost:5432/unittest?sslmode=disable"
	dbCfg := &database.Config{
		DSN:        endpoint,
		DriverName: "postgres",
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	}
	return database.InitDB(dbCfg)
}

func setupCoordinator(proversPerSession uint8, coordinatorURL string, nameForkMap map[string]int64) (*cron.Collector, *http.Server) {
	db, err := GetGormDBClient()
	if err != nil {
		panic(err.Error())
	}

	tokenTimeout := 6
	conf := &config.Config{
		L2: &config.L2{
			ChainID: 111,
		},
		ProverManager: &config.ProverManager{
			ProversPerSession: proversPerSession,
			Verifier: &config.VerifierConfig{
				MockMode: true,
			},
			BatchCollectionTimeSec: 10,
			ChunkCollectionTimeSec: 10,
			MaxVerifierWorkers:     10,
			SessionAttempts:        5,
			MinProverVersion:       version.Version,
		},
		Auth: &config.Auth{
			ChallengeExpireDurationSec: tokenTimeout,
			LoginExpireDurationSec:     tokenTimeout,
		},
	}

	var chainConf params.ChainConfig
	for forkName, forkNumber := range nameForkMap {
		switch forkName {
		case "shanghai":
			chainConf.ShanghaiBlock = big.NewInt(forkNumber)
		case "bernoulli":
			chainConf.BernoulliBlock = big.NewInt(forkNumber)
		case "london":
			chainConf.LondonBlock = big.NewInt(forkNumber)
		case "istanbul":
			chainConf.IstanbulBlock = big.NewInt(forkNumber)
		case "homestead":
			chainConf.HomesteadBlock = big.NewInt(forkNumber)
		case "eip155":
			chainConf.EIP155Block = big.NewInt(forkNumber)
		}
	}

	proofCollector := cron.NewCollector(context.Background(), db, conf, nil)

	router := gin.New()
	api.InitController(conf, &chainConf, db, nil)
	route.Route(router, conf, nil)
	srv := &http.Server{
		Addr:    coordinatorURL,
		Handler: router,
	}
	go func() {
		runErr := srv.ListenAndServe()
		if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			panic(runErr.Error())
		}
	}()
	time.Sleep(time.Second * 2)

	return proofCollector, srv
}

func main() {
	coordinatorURL := ":9091"
	nameForkMap := map[string]int64{"london": 2,
		"istanbul":  3,
		"bernoulli": 4}
	setupCoordinator(1, coordinatorURL, nameForkMap)

	var c = make(chan struct{}, 1)
	_ = <-c
}
