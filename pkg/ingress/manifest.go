// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ingress

import (
	"context"
	"reflect"

	"github.com/hashicorp/go-multierror"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func diffRoutes(olds, news []*apisixv1.Route) (added, updated, deleted []*apisixv1.Route) {
	if olds == nil {
		return news, nil, nil
	}
	if news == nil {
		return nil, nil, olds
	}

	oldMap := make(map[string]*apisixv1.Route, len(olds))
	newMap := make(map[string]*apisixv1.Route, len(news))
	for _, r := range olds {
		oldMap[r.ID] = r
	}
	for _, r := range news {
		newMap[r.ID] = r
	}

	for _, r := range news {
		if or, ok := oldMap[r.ID]; !ok {
			added = append(added, r)
		} else if !reflect.DeepEqual(or, r) {
			updated = append(updated, r)
		}
	}
	for _, r := range olds {
		if _, ok := newMap[r.ID]; !ok {
			deleted = append(deleted, r)
		}
	}
	return
}

func diffUpstreams(olds, news []*apisixv1.Upstream) (added, updated, deleted []*apisixv1.Upstream) {
	oldMap := make(map[string]*apisixv1.Upstream, len(olds))
	newMap := make(map[string]*apisixv1.Upstream, len(news))
	for _, u := range olds {
		oldMap[u.ID] = u
	}
	for _, u := range news {
		newMap[u.ID] = u
	}

	for _, u := range news {
		if ou, ok := oldMap[u.ID]; !ok {
			added = append(added, u)
		} else if !reflect.DeepEqual(ou, u) {
			updated = append(updated, u)
		}
	}
	for _, u := range olds {
		if _, ok := newMap[u.ID]; !ok {
			deleted = append(deleted, u)
		}
	}
	return
}

func diffStreamRoutes(olds, news []*apisixv1.StreamRoute) (added, updated, deleted []*apisixv1.StreamRoute) {
	oldMap := make(map[string]*apisixv1.StreamRoute, len(olds))
	newMap := make(map[string]*apisixv1.StreamRoute, len(news))
	for _, sr := range olds {
		oldMap[sr.ID] = sr
	}
	for _, sr := range news {
		newMap[sr.ID] = sr
	}

	for _, sr := range news {
		if ou, ok := oldMap[sr.ID]; !ok {
			added = append(added, sr)
		} else if !reflect.DeepEqual(ou, sr) {
			updated = append(updated, sr)
		}
	}
	for _, sr := range olds {
		if _, ok := newMap[sr.ID]; !ok {
			deleted = append(deleted, sr)
		}
	}
	return
}

type manifest struct {
	routes       []*apisixv1.Route
	upstreams    []*apisixv1.Upstream
	streamRoutes []*apisixv1.StreamRoute
}

func (m *manifest) diff(om *manifest) (added, updated, deleted *manifest) {
	ar, ur, dr := diffRoutes(om.routes, m.routes)
	au, uu, du := diffUpstreams(om.upstreams, m.upstreams)
	asr, usr, dsr := diffStreamRoutes(om.streamRoutes, m.streamRoutes)
	if ar != nil || au != nil || asr != nil {
		added = &manifest{
			routes:       ar,
			upstreams:    au,
			streamRoutes: asr,
		}
	}
	if ur != nil || uu != nil || usr != nil {
		updated = &manifest{
			routes:       ur,
			upstreams:    uu,
			streamRoutes: usr,
		}
	}
	if dr != nil || du != nil || dsr != nil {
		deleted = &manifest{
			routes:       dr,
			upstreams:    du,
			streamRoutes: dsr,
		}
	}
	return
}

func (c *Controller) syncManifests(ctx context.Context, added, updated, deleted *manifest) error {
	var merr *multierror.Error

	clusterName := c.cfg.APISIX.DefaultClusterName
	if deleted != nil {
		for _, r := range deleted.routes {
			if err := c.apisix.Cluster(clusterName).Route().Delete(ctx, r); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, u := range deleted.upstreams {
			if err := c.apisix.Cluster(clusterName).Upstream().Delete(ctx, u); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, sr := range deleted.streamRoutes {
			if err := c.apisix.Cluster(clusterName).StreamRoute().Delete(ctx, sr); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	if added != nil {
		// Should create upstreams firstly due to the dependencies.
		for _, u := range added.upstreams {
			if _, err := c.apisix.Cluster(clusterName).Upstream().Create(ctx, u); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, r := range added.routes {
			if _, err := c.apisix.Cluster(clusterName).Route().Create(ctx, r); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, sr := range added.streamRoutes {
			if _, err := c.apisix.Cluster(clusterName).StreamRoute().Create(ctx, sr); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	if updated != nil {
		for _, r := range updated.upstreams {
			if _, err := c.apisix.Cluster(clusterName).Upstream().Update(ctx, r); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, r := range updated.routes {
			if _, err := c.apisix.Cluster(clusterName).Route().Update(ctx, r); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, sr := range updated.streamRoutes {
			if _, err := c.apisix.Cluster(clusterName).StreamRoute().Create(ctx, sr); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	if merr != nil {
		return merr
	}
	return nil
}
