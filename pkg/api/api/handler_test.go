/*
Copyright 2021 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// mockDatabase database.db
type mockDatabase struct {
	database.Db
	clusters []registryv1.Cluster
}

func (m mockDatabase) GetCluster(name string) (*registryv1.Cluster, error) {
	for _, c := range m.clusters {
		if c.Spec.Name == name {
			return &c, nil
		}
	}
	return nil, nil
}

func (m mockDatabase) ListClusters(region string, environment string, businessUnit string, status string) ([]registryv1.Cluster, int, error) {
	return m.clusters, len(m.clusters), nil
}

func TestNewHandler(t *testing.T) {
	test := assert.New(t)
	d := mockDatabase{}
	m := monitoring.NewMetrics("cluster_registry_api_handler_test", nil, true)
	h := NewHandler(d, m)
	test.NotNil(h)
}

func TestGetCluster(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name             string
		clusterName      string
		clusters         []registryv1.Cluster
		expectedResponse string
		expectedStatus   int
	}{
		{
			name:        "get existing cluster",
			clusterName: "cluster1",
			clusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}}},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "get nonexistent cluster",
			clusterName: "cluster2",
			clusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}}},
			expectedStatus: http.StatusNotFound,
		},
	}
	for _, tc := range tcs {

		d := mockDatabase{clusters: tc.clusters}
		m := monitoring.NewMetrics("cluster_registry_api_handler_test", nil, true)
		h := NewHandler(d, m)
		r := NewRouter()

		req := httptest.NewRequest(echo.GET, "/api/v1/clusters/:name", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		ctx := r.NewContext(req, rec)
		ctx.SetPath("/api/articles/:name")
		ctx.SetParamNames("name")
		ctx.SetParamValues(tc.clusterName)

		err := h.GetCluster(ctx)
		test.NoError(err)

		test.Equal(tc.expectedStatus, rec.Code)

		if rec.Code == http.StatusOK {
			var c registryv1.ClusterSpec
			err := json.Unmarshal(rec.Body.Bytes(), &c)
			test.NoError(err)
			test.Equal(tc.clusterName, c.Name)
		}
	}
}

func TestListClusters(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name           string
		clusters       []registryv1.Cluster
		expectedStatus int
		expectedItems  int
	}{
		{
			name: "get all clusters",
			clusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}}, {
				Spec: registryv1.ClusterSpec{
					Name:         "cluster2",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}}},
			expectedStatus: http.StatusOK,
			expectedItems:  2,
		},
	}
	for _, tc := range tcs {

		d := mockDatabase{clusters: tc.clusters}
		m := monitoring.NewMetrics("cluster_registry_api_handler_test", nil, true)
		h := NewHandler(d, m)
		r := NewRouter()

		req := httptest.NewRequest(echo.GET, "/api/v1/clusters", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		ctx := r.NewContext(req, rec)

		err := h.ListClusters(ctx)

		test.NoError(err)
		test.Equal(tc.expectedStatus, rec.Code)

		if rec.Code == http.StatusOK {
			var cl clusterList
			err := json.Unmarshal(rec.Body.Bytes(), &cl)

			test.NoError(err)
			test.Equal(tc.expectedItems, cl.ItemsCount)
		}
	}
}
