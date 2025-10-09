package proxy

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"
	"scroll-tech/coordinator/internal/config"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// SubmitProofController the submit proof api controller
type SubmitProofController struct {
	proverMgr        *ProverManager
	clients          Clients
	priorityUpstream *PriorityUpstreamManager
}

// NewSubmitProofController create the submit proof api controller instance
func NewSubmitProofController(cfg *config.ProxyConfig, clients Clients, proverMgr *ProverManager, priorityMgr *PriorityUpstreamManager, reg prometheus.Registerer) *SubmitProofController {
	return &SubmitProofController{
		proverMgr:        proverMgr,
		clients:          clients,
		priorityUpstream: priorityMgr,
	}
}

func upstreamFromTaskName(taskID string) (string, string) {
	parts, rest, found := strings.Cut(taskID, ":")
	if found {
		return parts, rest
	}
	return "", parts
}

func formUpstreamWithTaskName(upstream string, taskID string) string {
	return fmt.Sprintf("%s:%s", upstream, taskID)
}

// SubmitProof prover submit the proof to coordinator
func (spc *SubmitProofController) SubmitProof(ctx *gin.Context) {

	var submitParameter coordinatorType.SubmitProofParameter
	if err := ctx.ShouldBind(&submitParameter); err != nil {
		nerr := fmt.Errorf("prover submitProof parameter invalid, err:%w", err)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	publicKey := getSessionData(ctx)
	if publicKey == "" {
		return
	}

	session := spc.proverMgr.Get(publicKey)
	upstream, realTaskID := upstreamFromTaskName(submitParameter.TaskID)
	cli, existed := spc.clients[upstream]
	if !existed {
		log.Warn("A upstream for submitting is removed or lost for some reason while running", "up", upstream)
		nerr := fmt.Errorf("Invalid upstream name (%s) from taskID %s", upstream, submitParameter.TaskID)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}
	log.Debug("Start submitting", "up", upstream, "cli", session.CliName, "id", realTaskID, "status", submitParameter.Status)
	submitParameter.TaskID = realTaskID

	resp, err := session.SubmitProof(ctx, &submitParameter, cli, upstream)
	if err != nil {
		log.Error("Upstream has error resp for submit", "code", resp.ErrCode, "msg", resp.ErrMsg, "up", upstream, "cli", session.CliName)
		types.RenderFailure(ctx, types.ErrCoordinatorGetTaskFailure, err)
		return
	} else if resp.ErrCode != 0 {
		log.Error("Upstream has error resp for get task", "code", resp.ErrCode, "msg", resp.ErrMsg, "up", upstream, "cli", session.CliName)
		// simply dispatch the error from upstream to prover
		types.RenderFailure(ctx, resp.ErrCode, fmt.Errorf("%s", resp.ErrMsg))
		return
	} else {
		log.Debug("Submit proof to upstream", "up", upstream, "cli", session.CliName, "taskID", realTaskID)
		spc.priorityUpstream.Delete(upstream)
		types.RenderSuccess(ctx, resp.Data)
		return
	}
}
