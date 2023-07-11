package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"scroll-tech/miner-api/internal/orm"
)

type Controller struct {
	engine *gin.Engine
	db     *orm.ProverTask
}

func NewController(db *orm.ProverTask) *Controller {
	r := gin.Default()
	v1 := r.Group("/api/v1")
	c := &Controller{
		db: db,
	}
	v1.GET("/tasks", c.GetTasksByProver)
	v1.GET("/total_rewards", c.GetTotalRewards)
	v1.GET("/task", c.GetTask)

	c.engine = r
	return c
}

func (c *Controller) Run(port string) {
	c.engine.Run(port)
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
// @Router       /tasks/{pubkey} [get]
func (c *Controller) GetTasksByProver(ctx *gin.Context) {
	pubkey := ctx.Param("pubkey")
	tasks, err := c.db.GetProverTasksByProver(context.Background(), pubkey)
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
// @Router       /total_rewards/{pubkey} [get]
func (c *Controller) GetTotalRewards(ctx *gin.Context) {
	pubkey := ctx.Param("pubkey")
	tasks, err := c.db.GetProverTasksByProver(context.Background(), pubkey)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	var rewards uint64
	for _, task := range tasks {
		rewards += task.Reward
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
// @Router       /task/{pubkey} [get]
func (c *Controller) GetTask(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	tasks, err := c.db.GetProverTasksByHashes(context.Background(), []string{taskID})
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	if len(tasks) > 0 {
		ctx.JSON(http.StatusOK, tasks[0])
	}
}
