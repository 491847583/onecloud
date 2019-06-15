// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package qcloud

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type SPrivateIpAddress struct {
	Description      string
	Primary          bool
	PrivateIpAddress string
	PublicIpAddress  string
	IsWanIpBlocked   bool
	State            string
}

type SNetworkInterface struct {
	VpcId                       string
	SubnetId                    string
	NetworkInterfaceId          string
	NetworkInterfaceName        string
	NetworkInterfaceDescription string
	GroupSet                    []string
	Primary                     bool
	MacAddress                  string
	State                       string
	CreatedTime                 time.Time
	Attachment                  string
	Zone                        string
	PrivateIpAddressSet         []SPrivateIpAddress
}

func (region *SRegion) GetNetworkInterfaces(interfaceIds []string, subnetId string, offset int, limit int) ([]SNetworkInterface, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := map[string]string{}
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	for idx, interfaceId := range interfaceIds {
		params[fmt.Sprintf("NetworkInterfaceIds.%d", idx)] = interfaceId
	}

	if len(subnetId) > 0 {
		params["Filters.0.Name"] = "subnet-id"
		params["Filters.0.Values.0"] = subnetId
	}
	body, err := region.vpcRequest("DescribeNetworkInterfaces", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeNetworkInterfaces")
	}

	interfaces := []SNetworkInterface{}
	err = body.Unmarshal(&interfaces, "NetworkInterfaceSet")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal.NetworkInterfaceSet")
	}
	total, _ := body.Float("TotalCount")
	return interfaces, int(total), nil
}

func (region *SRegion) DeleteNetworkInterface(interfaceId string) error {
	params := map[string]string{}
	params["Region"] = region.Region
	params["NetworkInterfaceId"] = interfaceId

	_, err := region.vpcRequest("DeleteNetworkInterface", params)
	if err != nil {
		return errors.Wrapf(err, "vpcRequest.DeleteNetworkInterface")
	}
	return nil
}
