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

package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerResourceBase struct {
	// 负载均衡ID
	LoadbalancerId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

type SLoadbalancerResourceBaseManager struct {
	SVpcResourceBaseManager
	SZoneResourceBaseManager
}

func ValidateLoadbalancerResourceInput(userCred mcclient.TokenCredential, input api.LoadbalancerResourceInput) (*SLoadbalancer, api.LoadbalancerResourceInput, error) {
	lbObj, err := LoadbalancerManager.FetchByIdOrName(userCred, input.Loadbalancer)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", LoadbalancerManager.Keyword(), input.Loadbalancer)
		} else {
			return nil, input, errors.Wrap(err, "LoadbalancerManager.FetchByIdOrName")
		}
	}
	input.Loadbalancer = lbObj.GetId()
	return lbObj.(*SLoadbalancer), input, nil
}

func (self *SLoadbalancerResourceBase) GetLoadbalancer() *SLoadbalancer {
	w, _ := LoadbalancerManager.FetchById(self.LoadbalancerId)
	if w != nil {
		return w.(*SLoadbalancer)
	}
	return nil
}

func (self *SLoadbalancerResourceBase) GetVpc() *SVpc {
	lb := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetVpc()
	}
	return nil
}

func (self *SLoadbalancerResourceBase) GetCloudprovider() *SCloudprovider {
	vpc := self.GetVpc()
	if vpc != nil {
		return vpc.GetCloudprovider()
	}
	return nil
}

func (self *SLoadbalancerResourceBase) GetCloudproviderId() string {
	cloudprovider := self.GetCloudprovider()
	if cloudprovider != nil {
		return cloudprovider.Id
	}
	return ""
}

func (self *SLoadbalancerResourceBase) GetProviderName() string {
	vpc := self.GetVpc()
	if vpc != nil {
		return vpc.GetProviderName()
	}
	return ""
}

func (self *SLoadbalancerResourceBase) GetCloudaccount() *SCloudaccount {
	vpc := self.GetVpc()
	if vpc != nil {
		return vpc.GetCloudaccount()
	}
	return nil
}

func (self *SLoadbalancerResourceBase) GetRegion() *SCloudregion {
	vpc := self.GetVpc()
	if vpc == nil {
		return nil
	}
	region, _ := vpc.GetRegion()
	return region
}

func (self *SLoadbalancerResourceBase) GetRegionId() string {
	region := self.GetRegion()
	if region != nil {
		return region.Id
	}
	return ""
}

func (self *SLoadbalancerResourceBase) GetZone() *SZone {
	lb := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetZone()
	}
	return nil
}

func (self *SLoadbalancerResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) api.LoadbalancerResourceInfo {
	return api.LoadbalancerResourceInfo{}
}

func (manager *SLoadbalancerResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerResourceInfo {
	rows := make([]api.LoadbalancerResourceInfo, len(objs))

	lbIds := make([]string, len(objs))
	for i := range objs {
		var base *SLoadbalancerResourceBase
		err := reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if err != nil {
			log.Errorf("Cannot find SLoadbalancerResourceBase in object %#v: %s", objs[i], err)
			continue
		}
		lbIds[i] = base.LoadbalancerId
	}

	lbs := make(map[string]SLoadbalancer)
	err := db.FetchStandaloneObjectsByIds(LoadbalancerManager, lbIds, &lbs)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}

	vpcList := make([]interface{}, len(rows))
	zoneList := make([]interface{}, len(rows))
	for i := range rows {
		rows[i] = api.LoadbalancerResourceInfo{}
		if lb, ok := lbs[lbIds[i]]; ok {
			rows[i].Loadbalancer = lb.Name
			rows[i].VpcId = lb.VpcId
			rows[i].ZoneId = lb.ZoneId
		}
		vpcList[i] = &SVpcResourceBase{rows[i].VpcId}
		zoneList[i] = &SZoneResourceBase{rows[i].ZoneId}
	}

	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, vpcList, fields, isList)
	zoneRows := manager.SZoneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, zoneList, fields, isList)

	for i := range rows {
		rows[i].VpcResourceInfo = vpcRows[i]
		rows[i].ZoneResourceInfoBase = zoneRows[i].ZoneResourceInfoBase
	}
	return rows
}

func (manager *SLoadbalancerResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.Loadbalancer) > 0 {
		lbObj, _, err := ValidateLoadbalancerResourceInput(userCred, query.LoadbalancerResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateLoadbalancerResourceInput")
		}
		q = q.Equals("loadbalancer_id", lbObj.GetId())
	}

	lbQ := LoadbalancerManager.Query("id").Snapshot()

	lbQ, err := manager.SVpcResourceBaseManager.ListItemFilter(ctx, lbQ, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}

	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	lbQ, err = manager.SZoneResourceBaseManager.ListItemFilter(ctx, lbQ, userCred, zoneQuery)
	if err != nil {
		return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemFilter")
	}

	if lbQ.IsAltered() {
		q = q.Filter(sqlchemy.In(q.Field("loadbalancer_id"), lbQ.SubQuery()))
	}
	return q, nil
}

func (manager *SLoadbalancerResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "loadbalancer" {
		lbQuery := LoadbalancerManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(lbQuery.Field("name", field))
		q = q.Join(lbQuery, sqlchemy.Equals(q.Field("loadbalancer_id"), lbQuery.Field("id")))
		q.GroupBy(lbQuery.Field("name"))
		return q, nil
	} else {
		lbs := LoadbalancerManager.Query("id", "zone_id", "vpc_id").SubQuery()
		q = q.LeftJoin(lbs, sqlchemy.Equals(q.Field("loadbalancer_id"), lbs.Field("id")))
		q, err := manager.SZoneResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
		if err == nil {
			return q, nil
		}

		return q, httperrors.ErrNotFound
	}
}

func (manager *SLoadbalancerResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerFilterListInput,
) (*sqlchemy.SQuery, error) {
	q, orders, fields := manager.GetOrderBySubQuery(q, userCred, query)
	if len(orders) > 0 {
		q = db.OrderByFields(q, orders, fields)
	}
	return q, nil
}

func (manager *SLoadbalancerResourceBaseManager) GetOrderBySubQuery(
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerFilterListInput,
) (*sqlchemy.SQuery, []string, []sqlchemy.IQueryField) {
	lbQ := LoadbalancerManager.Query("id", "name")
	var orders []string
	var fields []sqlchemy.IQueryField
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	if db.NeedOrderQuery(manager.SZoneResourceBaseManager.GetOrderByFields(zoneQuery)) {
		var zoneOrders []string
		var zoneFields []sqlchemy.IQueryField
		lbQ, zoneOrders, zoneFields = manager.SZoneResourceBaseManager.GetOrderBySubQuery(lbQ, userCred, zoneQuery)
		if len(zoneOrders) > 0 {
			orders = append(orders, zoneOrders...)
			fields = append(fields, zoneFields...)
		}
	}

	if db.NeedOrderQuery(manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)) {
		var vpcOrders []string
		var vpcFields []sqlchemy.IQueryField
		lbQ, vpcOrders, vpcFields = manager.SVpcResourceBaseManager.GetOrderBySubQuery(lbQ, userCred, query.VpcFilterListInput)
		if len(vpcOrders) > 0 {
			orders = append(orders, vpcOrders...)
			fields = append(fields, vpcFields...)
		}
	}

	if db.NeedOrderQuery(manager.GetOrderByFields(query)) {
		subq := lbQ.SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("loadbalancer_id"), subq.Field("id")))
		if db.NeedOrderQuery([]string{query.OrderByLoadbalancer}) {
			orders = append(orders, query.OrderByLoadbalancer)
			fields = append(fields, subq.Field("name"))
		}
	}

	return q, orders, fields
}

func (manager *SLoadbalancerResourceBaseManager) GetOrderByFields(query api.LoadbalancerFilterListInput) []string {
	fields := make([]string, 0)
	zoneQuery := api.ZonalFilterListInput{
		ZonalFilterListBase: query.ZonalFilterListBase,
	}
	zoneFields := manager.SZoneResourceBaseManager.GetOrderByFields(zoneQuery)
	fields = append(fields, zoneFields...)
	vpcFields := manager.SVpcResourceBaseManager.GetOrderByFields(query.VpcFilterListInput)
	fields = append(fields, vpcFields...)
	fields = append(fields, query.OrderByLoadbalancer)
	return fields
}

func (manager *SLoadbalancerResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		var err error
		subq := LoadbalancerManager.Query("id", "name", "vpc_id", "zone_id").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("loadbalancer_id"), q.Field("id")))
		if keys.Contains("loadbalancer") {
			q = q.AppendField(subq.Field("name", "loadbalancer"))
		}
		if keys.Contains("vpc") {
			q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
			}
		}
		if keys.ContainsAny(manager.SZoneResourceBaseManager.GetExportKeys()...) {
			q, err = manager.SZoneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
			if err != nil {
				return nil, errors.Wrap(err, "SZoneResourceBaseManager.ListItemExportKeys")
			}
		}
	}
	return q, nil
}

func (manager *SLoadbalancerResourceBaseManager) GetExportKeys() []string {
	keys := []string{"loadbalancer"}
	keys = append(keys, manager.SZoneResourceBaseManager.GetExportKeys()...)
	keys = append(keys, "vpc")
	return keys
}

func (self *SLoadbalancerResourceBase) GetChangeOwnerCandidateDomainIds() []string {
	lb := self.GetLoadbalancer()
	if lb != nil {
		return lb.GetChangeOwnerCandidateDomainIds()
	}
	return nil
}
