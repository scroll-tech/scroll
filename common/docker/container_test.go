package docker_test

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	cli *client.Client
)

func init() {
	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())
}

func TestPP(t *testing.T) {
	res, err := cli.ConfigCreate(context.Background(), swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   "zz",
			Labels: map[string]string{},
		},
		Data: []byte("const postgres config"),
	})
	assert.NoError(t, err)
	filter := filters.NewArgs()
	filter.Add("id", res.ID)
	lst, err := cli.ConfigList(context.Background(), types.ConfigListOptions{
		Filters: filter,
	})
	assert.NoError(t, err)
	t.Logf("ID: %s, version: %d\n", res.ID, lst[0].Version.Index)
}

func TestCC(t *testing.T) {
	//err := cli.ConfigUpdate(context.Background(), "zma9k3nsfcmt9saimdd42p9ii", swarm.Version{Index: 166}, swarm.ConfigSpec{
	//	Annotations: swarm.Annotations{
	//		Name:   "xx",
	//		Labels: map[string]string{"1": "3"},
	//	},
	//})
	//assert.NoError(t, err)

	filter := filters.NewArgs()
	filter.Add("name", "zz")
	lst, err := cli.ConfigList(context.Background(), types.ConfigListOptions{
		Filters: filter,
	})
	assert.NoError(t, err)
	for _, l := range lst {
		t.Log(l.ID, l.Spec.Labels, l.Spec)
		err = cli.ConfigRemove(context.Background(), l.ID)
		assert.NoError(t, err)
	}
}
