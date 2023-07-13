package controller

import (
	"net/http"
	"scroll-tech/prover-stats-api/internal/middleware"

	"scroll-tech/prover-stats-api/internal/logic"

	"github.com/gin-gonic/gin"
)

type ProverTaskController struct {
	router *gin.RouterGroup
	logic  *logic.ProverTaskLogic
}

func NewProverTaskController(r *gin.RouterGroup, taskLogic *logic.ProverTaskLogic) *ProverTaskController {
	router := r.Group("/prover_task")
	return &ProverTaskController{
		router: router,
		logic:  taskLogic,
	}
}

func (c *ProverTaskController) Route() {
	c.router.GET("/request_token", c.RequestToken)
	c.router.GET("/tasks", c.GetTasksByProver)
	c.router.GET("/total_rewards", c.GetTotalRewards)
	c.router.GET("/task", c.GetTask)
}

// RequestToken godoc
// @Summary      request token
// @Description  generate token for the client
// @Tags         prover_task
// @Produce      json
// @Success      200  {object}  string
// @Failure      401  {object}  string
// @Router       /prover_task/request_token [get]
func (c *ProverTaskController) RequestToken(ctx *gin.Context) {
	token, err := middleware.GenToken()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": token})
}

// GetTasksByProver godoc
// @Summary      give the proverTasks
// @Description  get []*ProverTask by prover public key
// @Tags         prover_task
// @Accept       string
// @Produce      json
// @Param        pubkey   path  string  true  "prover public key"
// @Success      200  {object}  []*orm.ProverTask
// @Failure      404  {object}  string
// @Failure      500  {object}  string
// @Router       /prover_task/tasks/{pubkey} [get]
func (c *ProverTaskController) GetTasksByProver(ctx *gin.Context) {
	pubkey := ctx.Query("pubkey")
	tasks, err := c.logic.GetTasksByProver(pubkey)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, tasks)
}

// GetTotalRewards godoc
// @Summary      give the total rewards of a prover
// @Description  get uint64 by prover public key
// @Tags         prover_task
// @Accept       string
// @Produce      json
// @Param        pubkey   path  string  true  "prover public key"
// @Success      200  {object}  map[string]*big.Int
// @Failure      404  {object}  string
// @Failure      500  {object}  string
// @Router       /prover_task/total_rewards/{pubkey} [get]
func (c *ProverTaskController) GetTotalRewards(ctx *gin.Context) {
	pubkey := ctx.Query("pubkey")
	rewards, err := c.logic.GetTotalRewards(pubkey)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"rewards": rewards})
}

// GetTask godoc
// @Summary      give the specific prover task
// @Description  get prover task by task id
// @Tags         prover_task
// @Accept       string
// @Produce      json
// @Param        task_id  path  string  true  "prover task hash"
// @Success      200  {object}  *orm.ProverTask
// @Failure      404  {object}  string
// @Failure      500  {object}  string
// @Router       /prover_task/task/{task_id} [get]
func (c *ProverTaskController) GetTask(ctx *gin.Context) {
	taskID := ctx.Query("task_id")
	task, err := c.logic.GetTask(taskID)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, task)
}
