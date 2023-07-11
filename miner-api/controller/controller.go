package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"scroll-tech/miner-api/service"
)

type ProverTaskController struct {
	router  *gin.RouterGroup
	service *service.ProverTaskService
}

func NewProverTaskController(r *gin.RouterGroup, taskService *service.ProverTaskService) *ProverTaskController {
	router := r.Group("/prover_task")
	return &ProverTaskController{
		router:  router,
		service: taskService,
	}
}

func (c *ProverTaskController) Route() {
	c.router.GET("/tasks", c.GetTasksByProver)
	c.router.GET("/total_rewards", c.GetTotalRewards)
	c.router.GET("/task", c.GetTask)
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
	pubkey := ctx.Param("pubkey")
	tasks, err := c.service.GetTasksByProver(pubkey)
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
// @Success      200  {object}  map[string]uint64
// @Failure      404  {object}  string
// @Failure      500  {object}  string
// @Router       /prover_task/total_rewards/{pubkey} [get]
func (c *ProverTaskController) GetTotalRewards(ctx *gin.Context) {
	pubkey := ctx.Param("pubkey")
	rewards, err := c.service.GetTotalRewards(pubkey)
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
	taskID := ctx.Param("task_id")
	task, err := c.service.GetTask(taskID)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, task)

}
