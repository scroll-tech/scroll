package proxy

import (
	"fmt"
	"math/rand"
	"sync"

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

// PriorityUpstreamManager manages priority upstream mappings with thread safety
type PriorityUpstreamManager struct {
	sync.RWMutex
	data map[string]string
}

// NewPriorityUpstreamManager creates a new PriorityUpstreamManager
func NewPriorityUpstreamManager() *PriorityUpstreamManager {
	return &PriorityUpstreamManager{
		data: make(map[string]string),
	}
}

// Get retrieves the priority upstream for a given key
func (p *PriorityUpstreamManager) Get(key string) (string, bool) {
	p.RLock()
	defer p.RUnlock()
	value, exists := p.data[key]
	return value, exists
}

// Set sets the priority upstream for a given key
func (p *PriorityUpstreamManager) Set(key, value string) {
	p.Lock()
	defer p.Unlock()
	p.data[key] = value
}

// Delete removes the priority upstream for a given key
func (p *PriorityUpstreamManager) Delete(key string) {
	p.Lock()
	defer p.Unlock()
	delete(p.data, key)
}

// GetTaskController the get prover task api controller
type GetTaskController struct {
	proverMgr        *ProverManager
	clients          Clients
	priorityUpstream *PriorityUpstreamManager

	workingRnd           *rand.Rand
	getTaskAccessCounter *prometheus.CounterVec
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.ProxyConfig, clients Clients, proverMgr *ProverManager, priorityMgr *PriorityUpstreamManager, reg prometheus.Registerer) *GetTaskController {
	// TODO: implement proxy get task controller initialization
	return &GetTaskController{
		priorityUpstream: priorityMgr,
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
				ptc.priorityUpstream.Set(publicKey, upStream)
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
	priorityUpstream, exist := ptc.priorityUpstream.Get(publicKey)
	if exist {
		cli := ptc.clients[priorityUpstream]
		if cli != nil && !getTask(priorityUpstream, cli) {
			return
		} else if cli == nil {
			// TODO: log error
		}
	}

	// Create a slice to hold the keys
	keys := make([]string, 0, len(ptc.clients))
	for k := range ptc.clients {
		keys = append(keys, k)
	}

	// Shuffle the keys using a local RNG (avoid deprecated rand.Seed)
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	// Iterate over the shuffled keys
	for _, n := range keys {
		if !getTask(n, ptc.clients[n]) {
			return
		}
	}

	// if all get task failed, throw empty proof resp
	types.RenderFailure(ctx, types.ErrCoordinatorEmptyProofData, fmt.Errorf("get empty prover task"))
}
