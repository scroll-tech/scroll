package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"

	"scroll-tech/coordinator/internal/config"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

func getSessionData(ctx *gin.Context) (string, *coordinatorType.LoginParameter) {

	publicKeyData, publicKeyExist := ctx.Get(coordinatorType.PublicKey)
	publicKey, castOk := publicKeyData.(string)
	if !publicKeyExist || !castOk {
		nerr := fmt.Errorf("no public key binding: %v", publicKeyData)
		log.Warn("get_task parameter fail", "error", nerr)

		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return "", nil
	}

	loginParamData, publicKeyExist := ctx.Get(LoginParamCache)
	loginParam, castOk := loginParamData.(*coordinatorType.LoginParameter)
	if !publicKeyExist || !castOk {
		nerr := fmt.Errorf("no login param binding: %v", loginParamData)
		log.Warn("get_task parameter fail", "error", nerr)

		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return "", nil
	}

	return publicKey, loginParam
}

// GetTaskController the get prover task api controller
type GetTaskController struct {
	tokenCache       *UserTokenCache
	clients          Clients
	priorityUpstream map[string]string

	getTaskAccessCounter *prometheus.CounterVec
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.Config, clients Clients, tokenCache *UserTokenCache, reg prometheus.Registerer) *GetTaskController {
	// TODO: implement proxy get task controller initialization
	return &GetTaskController{
		priorityUpstream: make(map[string]string),
		tokenCache:       tokenCache,
		clients:          clients,
	}
}

func (ptc *GetTaskController) incGetTaskAccessCounter(ctx *gin.Context) error {
	// TODO: implement proxy get task access counter
	return nil
}

// GetTasks get assigned chunk/batch task
func (ptc *GetTaskController) GetTasks(ctx *gin.Context) {
	var getTaskParameter coordinatorType.GetTaskParameter
	if err := ctx.ShouldBind(&getTaskParameter); err != nil {
		nerr := fmt.Errorf("prover task parameter invalid, err:%w", err)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	publicKey, loginParam := getSessionData(ctx)
	if publicKey == "" || loginParam == nil {
		return
	}

	tokens := ptc.tokenCache.Get(publicKey)

	onClientFail := func(upstream string) {
		//TODO: log re-connect request in info level

		request := TokenUpdate{
			PublicKey:      publicKey,
			Upstream:       upstream,
			Phase:          tokens.LoginPhase,
			LoginParam:     *loginParam,
			CompleteNotify: nil,
		}
		select {
		case <-ctx.Done():
		case ptc.tokenCache.tokenCacheUpdate <- &request:
		}

	}

	priorityUpstream, exist := ptc.priorityUpstream[publicKey]
	if exist {
		cli := ptc.clients[priorityUpstream]
		loginSchema := tokens.LoginData[priorityUpstream]
		if loginSchema == nil {
			onClientFail(priorityUpstream)
		} else {
			ret, triggerUpdate := getTaskFromClient(ctx, cli, &getTaskParameter, loginSchema.Token)
			if ret != nil {

			} else if triggerUpdate {
				onClientFail(priorityUpstream)
			}
		}
		types.RenderFailure(ctx, types.ErrCoordinatorEmptyProofData, fmt.Errorf("get empty prover task"))
	}

	for n, cli := range ptc.clients {

	}
}

func getTaskFromClient(ctx *gin.Context, cli Client, param *coordinatorType.GetTaskParameter, token string) (*coordinatorType.GetTaskSchema, bool) {

	theCli := cli.PeekClient()
	if theCli == nil {
		return nil, true
	}

	resp, err := theCli.GetTask(ctx, param, token)
	if err != nil {
		// log the err in error level
		return nil, false
	}

	// Parse response
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		unAuth := resp.StatusCode == http.StatusUnauthorized
		var respWithData types.Response
		// Note: Body is consumed after decoding, caller should not read it again
		if err := json.NewDecoder(resp.Body).Decode(&respWithData); err == nil {
			if unAuth && respWithData.ErrCode == types.ErrJWTTokenExpired {
				return nil, true
			}

			var getTaskResult coordinatorType.GetTaskSchema
			err = mapstructure.Decode(respWithData.Data, &getTaskResult)
			if err != nil {
				log.Error("parse get task data fail", "respdata", respWithData.Data)
				return nil, false
			}
			return &getTaskResult, false
		} else {
			log.Error("parse get task response failed", "error", err)
			//fmt.Errorf("login parsing response failed: %v", err)
			return nil, false
		}
	}

	return nil, false
}
