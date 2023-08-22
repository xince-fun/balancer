package controller

import exposerv1 "github.com/xince-fun/balancer/pkg/apis/balancer/v1"

func NewPodLabels(balancer *exposerv1.Balancer) map[string]string {
	return map[string]string{
		exposerv1.BalancerKey: balancer.Name,
	}
}

func NewServiceLabels(balancer *exposerv1.Balancer) map[string]string {
	return map[string]string{
		exposerv1.BalancerKey: balancer.Name,
	}
}
