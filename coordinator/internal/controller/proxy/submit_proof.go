package proxy

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"scroll-tech/common/types"
	"scroll-tech/coordinator/internal/config"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// SubmitProofController the submit proof api controller
type SubmitProofController struct {
	proverMgr *ProverManager
	clients   Clients
}

// NewSubmitProofController create the submit proof api controller instance
func NewSubmitProofController(cfg *config.ProxyConfig, clients Clients, proverMgr *ProverManager, reg prometheus.Registerer) *SubmitProofController {
	return &SubmitProofController{
		proverMgr: proverMgr,
		clients:   clients,
	}
}

func upstreamFromTaskName(taskID string) string {
	// TODO
	return ""
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
	upstream := upstreamFromTaskName(submitParameter.TaskID)
	cli, existed := spc.clients[upstream]
	if !existed {
		// TODO: log error
		nerr := fmt.Errorf("Invalid upstream name (%s) from taskID %s", upstream, submitParameter.TaskID)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	resp, err := session.SubmitProof(ctx, &submitParameter, cli, upstream)
	if err != nil {
		types.RenderFailure(ctx, types.ErrCoordinatorGetTaskFailure, err)
		return
	} else if resp.ErrCode != 0 {
		// simply dispatch the error from upstream to prover
		types.RenderFailure(ctx, resp.ErrCode, fmt.Errorf("%s", resp.ErrMsg))
		return
	} else {
		types.RenderSuccess(ctx, resp.Data)
		return
	}
}
