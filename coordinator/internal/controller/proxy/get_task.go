package proxy

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"

	"scroll-tech/coordinator/internal/config"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

func getSessionData(ctx *gin.Context) string {

	publicKeyData, publicKeyExist := ctx.Get(coordinatorType.PublicKey)
	publicKey, castOk := publicKeyData.(string)
	if !publicKeyExist || !castOk {
		nerr := fmt.Errorf("no public key binding: %v", publicKeyData)
		log.Warn("get_task parameter fail", "error", nerr)

		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return ""
	}

	return publicKey
}

// GetTaskController the get prover task api controller
type GetTaskController struct {
	proverMgr        *ProverManager
	clients          Clients
	priorityUpstream map[string]string

	getTaskAccessCounter *prometheus.CounterVec
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.ProxyConfig, clients Clients, proverMgr *ProverManager, reg prometheus.Registerer) *GetTaskController {
	// TODO: implement proxy get task controller initialization
	return &GetTaskController{
		priorityUpstream: make(map[string]string),
		proverMgr:        proverMgr,
		clients:          clients,
	}
}

func (ptc *GetTaskController) incGetTaskAccessCounter(ctx *gin.Context) error {
	// TODO: implement proxy get task access counter
	return nil
}

// GetTasks get assigned chunk/batch task
func (ptc *GetTaskController) GetTasks(ctx *gin.Context) {
	fmt.Println("start get task")
	var getTaskParameter coordinatorType.GetTaskParameter
	if err := ctx.ShouldBind(&getTaskParameter); err != nil {
		nerr := fmt.Errorf("prover task parameter invalid, err:%w", err)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	publicKey := getSessionData(ctx)
	if publicKey == "" {
		return
	}

	session := ptc.proverMgr.Get(publicKey)

	getTask := func(upStream string, cli Client) (tryNext bool) {
		resp, err := session.GetTask(ctx, &getTaskParameter, cli, upStream)
		fmt.Println("upstream get task", resp)
		if err != nil {
			types.RenderFailure(ctx, types.ErrCoordinatorGetTaskFailure, err)
			return
		} else if resp.ErrCode != types.ErrCoordinatorEmptyProofData {

			if resp.ErrCode != 0 {
				// simply dispatch the error from upstream to prover
				types.RenderFailure(ctx, resp.ErrCode, fmt.Errorf("%s", resp.ErrMsg))
				return
			}

			var task coordinatorType.GetTaskSchema
			if err = resp.DecodeData(&task); err == nil {
				task.TaskID = formUpstreamWithTaskName(upStream, task.TaskID)
				// TODO: log the new id in debug level
				types.RenderSuccess(ctx, &task)
			} else {
				types.RenderFailure(ctx, types.InternalServerError, fmt.Errorf("decode task fail: %v", err))
			}

			return
		}
		tryNext = true
		return
	}

	// if the priority upsteam is set, we try this upstream first until get the task resp or no task resp
	priorityUpstream, exist := ptc.priorityUpstream[publicKey]
	if exist {
		cli := ptc.clients[priorityUpstream]
		if cli != nil && !getTask(priorityUpstream, cli) {
			return
		} else if cli == nil {
			// TODO: log error
		}
	}

	for n, cli := range ptc.clients {
		if !getTask(n, cli) {
			return
		}
	}

	// if all get task failed, throw empty proof resp
	types.RenderFailure(ctx, types.ErrCoordinatorEmptyProofData, fmt.Errorf("get empty prover task"))
}
