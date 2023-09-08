package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	ctype "scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/prover-stats-api/internal/logic"
	"scroll-tech/prover-stats-api/internal/types"
)

// ProverTaskController provides API controller.
type ProverTaskController struct {
	logic *logic.ProverTaskLogic
}

// NewProverTaskController provides a ProverTask instance.
func NewProverTaskController(db *gorm.DB) *ProverTaskController {
	return &ProverTaskController{
		logic: logic.NewProverTaskLogic(db),
	}
}

// ProverTasks godoc
// @Summary    	 get all the prover task by prover public key
// @Description  get all the prover task by prover public key
// @Tags         prover_task
// @Accept       plain
// @Produce      plain
// @Param        pubkey   query  string  true  "prover public key"
// @Param        page     query  int 	 true  "page"
// @Param        page_size query  int 	 true  "page_size"
// @Param        Authorization header string false "Bearer license"
// @Success      200  {array}   types.ProverTaskSchema
// @Router       /api/prover_task/v1/tasks [get]
func (c *ProverTaskController) ProverTasks(ctx *gin.Context) {
	var pp types.ProverTasksPaginationParameter
	if err := ctx.ShouldBind(&pp); err != nil {
		nerr := fmt.Errorf("parameter invalid, err:%w", err)
		ctype.RenderFailure(ctx, types.ErrParameterInvalidNo, nerr)
		return
	}

	tasks, err := c.logic.GetTasksByProver(ctx, pp.PublicKey, pp.Page, pp.PageSize)
	if err != nil {
		nerr := fmt.Errorf("controller.ProverTasks err:%w", err)
		ctype.RenderFailure(ctx, types.ErrProverTaskFailure, nerr)
		return
	}

	var proverTaskSchemas []types.ProverTaskSchema
	for _, task := range tasks {
		proverTaskSchema := types.ProverTaskSchema{
			TaskID:        task.TaskID,
			ProverName:    task.ProverName,
			TaskType:      message.ProofType(task.TaskType).String(),
			ProvingStatus: ctype.ProvingStatus(task.ProvingStatus).String(),
			Reward:        task.Reward.String(),
			CreatedAt:     task.CreatedAt,
		}
		proverTaskSchemas = append(proverTaskSchemas, proverTaskSchema)
	}

	ctype.RenderSuccess(ctx, proverTaskSchemas)
}

// GetTotalRewards godoc
// @Summary      give the total rewards of a prover
// @Description  get uint64 by prover public key
// @Tags         prover_task
// @Accept       plain
// @Produce      plain
// @Param        pubkey   path  string  true  "prover public key"
// @Param        Authorization header string false "Bearer license"
// @Success      200  {object} types.ProverTotalRewardsSchema
// @Router       /api/prover_task/v1/total_rewards [get]
func (c *ProverTaskController) GetTotalRewards(ctx *gin.Context) {
	var pp types.ProverTotalRewardsParameter
	if err := ctx.ShouldBind(&pp); err != nil {
		nerr := fmt.Errorf("parameter invalid, err:%w", err)
		ctype.RenderFailure(ctx, types.ErrParameterInvalidNo, nerr)
		return
	}

	rewards, err := c.logic.GetTotalRewards(ctx, pp.PublicKey)
	if err != nil {
		nerr := fmt.Errorf("controller.GetTotalRewards, err:%w", err)
		ctype.RenderFailure(ctx, types.ErrProverTotalRewardFailure, nerr)
		return
	}

	resp := types.ProverTotalRewardsSchema{
		Rewards: rewards.String(),
	}

	ctype.RenderSuccess(ctx, resp)
}

// GetTask godoc
// @Summary      give the specific prover task
// @Description  get prover task by task id
// @Tags         prover_task
// @Accept       plain
// @Produce      plain
// @Param        task_id  path  string  true  "prover task hash"
// @Param        Authorization header string false "Bearer license"
// @Success      200  {object}  types.ProverTaskSchema
// @Router       /api/prover_task/v1/task [get]
func (c *ProverTaskController) GetTask(ctx *gin.Context) {
	var pp types.ProverTaskParameter
	if err := ctx.ShouldBind(&pp); err != nil {
		nerr := fmt.Errorf("parameter invalid, err:%w", err)
		ctype.RenderFailure(ctx, types.ErrParameterInvalidNo, nerr)
		return
	}

	task, err := c.logic.GetTask(ctx, pp.TaskID)
	if err != nil {
		nerr := fmt.Errorf("controller.GetTask, err:%w", err)
		ctype.RenderFailure(ctx, types.ErrProverTotalRewardFailure, nerr)
		return
	}

	schema := types.ProverTaskSchema{
		TaskID:        task.TaskID,
		ProverName:    task.ProverName,
		TaskType:      message.ProofType(task.TaskType).String(),
		ProvingStatus: ctype.ProvingStatus(task.ProvingStatus).String(),
		Reward:        task.Reward.String(),
		CreatedAt:     task.CreatedAt,
	}

	ctype.RenderSuccess(ctx, schema)
}
