// Copyright (c) 2021 Terminus, Inc.
//
// This program is free software: you can use, redistribute, and/or modify
// it under the terms of the GNU Affero General Public License, version 3
// or later ("AGPL"), as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package redis

import (
	"github.com/erda-project/erda/apistructs"
	"github.com/erda-project/erda/modules/scheduler/executor/plugins/k8s/addon"
	"github.com/erda-project/erda/modules/scheduler/executor/plugins/k8s/addon/redis/legacy"
	"github.com/erda-project/erda/pkg/httpclient"
)

type UnifiedRedisOperator struct {
	redisoperator       *RedisOperator
	legacyRedisoperator *legacy.RedisOperator
	useLegacy           bool
}

func New(k8sutil addon.K8SUtil,
	deploy addon.DeploymentUtil,
	sts addon.StatefulsetUtil,
	service addon.ServiceUtil,
	ns addon.NamespaceUtil,
	overcommit addon.OvercommitUtil,
	secret addon.SecretUtil,
	client *httpclient.HTTPClient) *UnifiedRedisOperator {
	return &UnifiedRedisOperator{
		redisoperator:       NewRedisOperator(k8sutil, deploy, sts, service, ns, overcommit, secret, client),
		legacyRedisoperator: legacy.New(k8sutil, deploy, sts, service, ns, overcommit, client),
		useLegacy:           false,
	}
}

func (ro *UnifiedRedisOperator) IsSupported() bool {
	if ro.redisoperator.IsSupported() {
		ro.useLegacy = false
		return true
	}
	if ro.legacyRedisoperator.IsSupported() {
		ro.useLegacy = true
		return true
	}
	return false
}

func (ro *UnifiedRedisOperator) Validate(sg *apistructs.ServiceGroup) error {
	if ro.useLegacy {
		return ro.legacyRedisoperator.Validate(sg)
	}
	return ro.redisoperator.Validate(sg)
}
func (ro *UnifiedRedisOperator) Convert(sg *apistructs.ServiceGroup) interface{} {
	if ro.useLegacy {
		return ro.legacyRedisoperator.Convert(sg)
	}
	return ro.redisoperator.Convert(sg)
}
func (ro *UnifiedRedisOperator) Create(k8syml interface{}) error {
	if ro.useLegacy {
		return ro.legacyRedisoperator.Create(k8syml)
	}
	return ro.redisoperator.Create(k8syml)
}

func (ro *UnifiedRedisOperator) Inspect(sg *apistructs.ServiceGroup) (*apistructs.ServiceGroup, error) {
	if ro.useLegacy {
		return ro.legacyRedisoperator.Inspect(sg)
	}
	return ro.redisoperator.Inspect(sg)
}
func (ro *UnifiedRedisOperator) Remove(sg *apistructs.ServiceGroup) error {
	if ro.useLegacy {
		return ro.legacyRedisoperator.Remove(sg)
	}
	if err := ro.redisoperator.Remove(sg); err != nil {
		return ro.legacyRedisoperator.Remove(sg)
	}
	return nil
}

func (ro *UnifiedRedisOperator) Update(k8syml interface{}) error {
	if ro.useLegacy {
		return ro.legacyRedisoperator.Update(k8syml)
	}
	return ro.redisoperator.Update(k8syml)
}
