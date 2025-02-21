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

package mns

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	"github.com/erda-project/erda/apistructs"
	"github.com/erda-project/erda/pkg/encrypt"
)

func (m *Mns) GetInstancesInfo(req apistructs.EcsInfoReq) (*ecs.DescribeInstancesResponse, error) {
	client, err := ecs.NewClientWithAccessKey(req.Region, req.AccessKeyId, req.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	if req.InstanceIds == nil {
		return nil, errors.New("empty instance ids")
	}

	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"
	request.RegionId = req.Region
	if req.InstanceIds != nil {
		content, err := json.Marshal(req.InstanceIds)
		if err != nil {
			errStr := fmt.Sprintf("json marshal error: %v", err)
			return nil, errors.New(errStr)
		}
		request.InstanceIds = string(content)
		request.PageSize = requests.Integer(strconv.Itoa(len(req.InstanceIds)))
	}

	// if not provide valid instance id, it will return other instance info by default pageNum & pageSize
	response, err := client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}
	if response.BaseResponse == nil {
		return nil, errors.New("base response in empty")
	}

	if !response.BaseResponse.IsSuccess() {
		errStr := fmt.Sprintf("base response status code: %d", response.BaseResponse.GetHttpStatus())
		return nil, errors.New(errStr)
	}

	return response, nil
}

func (m *Mns) GetInstancesPrivateIp(req apistructs.EcsInfoReq) (map[string]string, error) {
	result := make(map[string]string)
	if req.InstanceIds == nil {
		return nil, errors.New("empty instance ids")
	}
	response, err := m.GetInstancesInfo(req)
	if err != nil {
		return nil, err
	}
	validNum := 0
	for _, instance := range response.Instances.Instance {
		if instance.VpcAttributes.PrivateIpAddress.IpAddress == nil {
			return nil, errors.New("get empty instance private ip")
		}
		exist, err := contains(req.InstanceIds, instance.InstanceId)
		if err != nil {
			return nil, err
		}
		if !exist {
			errStr := fmt.Sprintf("instance id: %s, not in request instance ids: %v", instance.InstanceId, req.InstanceIds)
			return nil, errors.New(errStr)
		}
		result[instance.InstanceId] = instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
		validNum++
	}
	if validNum != len(req.InstanceIds) {
		errStr := fmt.Sprintf("valid instance num: %d, total num: %d, response: %v, all instance ids: %v", validNum, len(req.InstanceIds), result, req.InstanceIds)
		return nil, errors.New(errStr)
	}
	return result, nil

}

func (m *Mns) GetInstancesIDByPrivateIp(req apistructs.EcsInfoReq) (string, error) {
	accessKey := encrypt.AesDecrypt(req.AccessKeyId, apistructs.TerraformEcyKey)
	secretKey := encrypt.AesDecrypt(req.AccessKeySecret, apistructs.TerraformEcyKey)
	client, err := ecs.NewClientWithAccessKey(req.Region, accessKey, secretKey)
	if err != nil {
		return "", err
	}
	if req.PrivateIPs == nil {
		return "", errors.New("empty instance ids")
	}
	content, err := json.Marshal(req.PrivateIPs)
	if err != nil {
		errStr := fmt.Sprintf("json marshal error: %v", err)
		return "", errors.New(errStr)
	}
	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"

	request.PrivateIpAddresses = string(content)

	response, err := client.DescribeInstances(request)
	if err != nil {
		errStr := fmt.Sprintf("failed get id by private ip: %v\n", err)
		return "", errors.New(errStr)
	}
	if len(response.Instances.Instance) == 0 {
		errStr := fmt.Sprintf("do not find instance id by private ip: %v\n", err)
		return "", errors.New(errStr)
	}
	return response.Instances.Instance[0].InstanceId, nil
}
func contains(slice interface{}, item interface{}) (bool, error) {
	s := reflect.ValueOf(slice)

	if s.Kind() != reflect.Slice {
		return false, errors.New("invalid data type")
	}

	for i := 0; i < s.Len(); i++ {
		if s.Index(i).Interface() == item {
			return true, nil
		}
	}
	return false, nil
}
