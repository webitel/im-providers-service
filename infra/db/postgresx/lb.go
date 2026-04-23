package postgresx

import (
	"math/rand/v2"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LoadBalancingPolicy string

const (
	RoundRobin LoadBalancingPolicy = "round-robin"
	Random     LoadBalancingPolicy = "random"
)

type LoadBalancer interface {
	Resolve(pools []*pgxpool.Pool) *pgxpool.Pool
	Policy() LoadBalancingPolicy
}

type RoundRobinLoadBalancer struct{ counter atomic.Uint64 }

func (lb *RoundRobinLoadBalancer) Policy() LoadBalancingPolicy { return RoundRobin }
func (lb *RoundRobinLoadBalancer) Resolve(pools []*pgxpool.Pool) *pgxpool.Pool {
	n := uint64(len(pools))
	if n == 1 {
		return pools[0]
	}
	idx := lb.counter.Add(1) % n
	return pools[idx]
}

type RandomLoadBalancer struct{}

func (lb *RandomLoadBalancer) Policy() LoadBalancingPolicy { return Random }
func (lb *RandomLoadBalancer) Resolve(pools []*pgxpool.Pool) *pgxpool.Pool {
	if len(pools) == 1 {
		return pools[0]
	}
	return pools[rand.IntN(len(pools))]
}

func newLoadBalancer(policy ...LoadBalancingPolicy) LoadBalancer {
	if len(policy) == 0 {
		return &RoundRobinLoadBalancer{}
	}

	switch policy[0] {
	case Random:
		return &RandomLoadBalancer{}
	case RoundRobin:
		return &RoundRobinLoadBalancer{}
	default:
		panic("unexpected postgresx.LoadBalancingPolicy")
	}
}
