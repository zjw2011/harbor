// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/harbor/src/common"
	"github.com/vmware/harbor/src/common/models"
)

var (
	labelAPIBasePath = "/api/labels"
	labelID          int64
)

func TestLabelAPIPost(t *testing.T) {
	postFunc := func(resp *httptest.ResponseRecorder) error {
		id, err := parseResourceID(resp)
		if err != nil {
			return err
		}
		labelID = id
		return nil
	}

	cases := []*codeCheckingCase{
		// 401
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPost,
				url:    labelAPIBasePath,
			},
			code: http.StatusUnauthorized,
		},

		// 400
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodPost,
				url:        labelAPIBasePath,
				bodyJSON:   &models.Label{},
				credential: nonSysAdmin,
			},
			code: http.StatusBadRequest,
		},

		// 403 non-sysadmin try to create global label
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPost,
				url:    labelAPIBasePath,
				bodyJSON: &models.Label{
					Name:  "test",
					Scope: common.LabelScopeGlobal,
				},
				credential: nonSysAdmin,
			},
			code: http.StatusForbidden,
		},

		// 403 non-member user try to create project label
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPost,
				url:    labelAPIBasePath,
				bodyJSON: &models.Label{
					Name:      "test",
					Scope:     common.LabelScopeProject,
					ProjectID: 1,
				},
				credential: nonSysAdmin,
			},
			code: http.StatusForbidden,
		},

		// 403 developer try to create project label
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPost,
				url:    labelAPIBasePath,
				bodyJSON: &models.Label{
					Name:      "test",
					Scope:     common.LabelScopeProject,
					ProjectID: 1,
				},
				credential: projDeveloper,
			},
			code: http.StatusForbidden,
		},

		// 404 non-exist project
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPost,
				url:    labelAPIBasePath,
				bodyJSON: &models.Label{
					Name:      "test",
					Scope:     common.LabelScopeProject,
					ProjectID: 10000,
				},
				credential: projAdmin,
			},
			code: http.StatusNotFound,
		},

		// 200
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPost,
				url:    labelAPIBasePath,
				bodyJSON: &models.Label{
					Name:      "test",
					Scope:     common.LabelScopeProject,
					ProjectID: 1,
				},
				credential: projAdmin,
			},
			code:     http.StatusCreated,
			postFunc: postFunc,
		},

		// 409
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPost,
				url:    labelAPIBasePath,
				bodyJSON: &models.Label{
					Name:      "test",
					Scope:     common.LabelScopeProject,
					ProjectID: 1,
				},
				credential: projAdmin,
			},
			code: http.StatusConflict,
		},
	}

	runCodeCheckingCases(t, cases...)
}

func TestLabelAPIGet(t *testing.T) {
	cases := []*codeCheckingCase{
		// 400
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%d", labelAPIBasePath, 0),
			},
			code: http.StatusBadRequest,
		},

		// 404
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%d", labelAPIBasePath, 1000),
			},
			code: http.StatusNotFound,
		},

		// 200
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
			},
			code: http.StatusOK,
		},
	}
	runCodeCheckingCases(t, cases...)
}

func TestLabelAPIList(t *testing.T) {
	cases := []*codeCheckingCase{
		// 400 no scope query string
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodGet,
				url:    labelAPIBasePath,
			},
			code: http.StatusBadRequest,
		},

		// 400 invalid scope
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodGet,
				url:    labelAPIBasePath,
				queryStruct: struct {
					Scope string `url:"scope"`
				}{
					Scope: "invalid_scope",
				},
			},
			code: http.StatusBadRequest,
		},

		// 400 invalid project_id
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodGet,
				url:    labelAPIBasePath,
				queryStruct: struct {
					Scope     string `url:"scope"`
					ProjectID int64  `url:"project_id"`
				}{
					Scope:     "p",
					ProjectID: 0,
				},
			},
			code: http.StatusBadRequest,
		},
	}
	runCodeCheckingCases(t, cases...)

	// 200
	labels := []*models.Label{}
	err := handleAndParse(&testingRequest{
		method: http.MethodGet,
		url:    labelAPIBasePath,
		queryStruct: struct {
			Scope     string `url:"scope"`
			ProjectID int64  `url:"project_id"`
			Name      string `url:"name"`
		}{
			Scope:     "p",
			ProjectID: 1,
			Name:      "test",
		},
	}, &labels)
	require.Nil(t, err)
	assert.Equal(t, 1, len(labels))

	err = handleAndParse(&testingRequest{
		method: http.MethodGet,
		url:    labelAPIBasePath,
		queryStruct: struct {
			Scope     string `url:"scope"`
			ProjectID int64  `url:"project_id"`
			Name      string `url:"name"`
		}{
			Scope:     "p",
			ProjectID: 1,
			Name:      "dev",
		},
	}, &labels)
	require.Nil(t, err)
	assert.Equal(t, 0, len(labels))
}

func TestLabelAPIPut(t *testing.T) {
	cases := []*codeCheckingCase{
		// 401
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPut,
				url:    fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
			},
			code: http.StatusUnauthorized,
		},

		// 400
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodPut,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, 0),
				credential: nonSysAdmin,
			},
			code: http.StatusBadRequest,
		},

		// 404
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodPut,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, 10000),
				credential: nonSysAdmin,
			},
			code: http.StatusNotFound,
		},

		// 403 non-member user
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodPut,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				credential: nonSysAdmin,
			},
			code: http.StatusForbidden,
		},

		// 403 developer
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodPut,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				credential: projDeveloper,
			},
			code: http.StatusForbidden,
		},

		// 400
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPut,
				url:    fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				bodyJSON: &models.Label{
					Name:      "",
					Scope:     common.LabelScopeProject,
					ProjectID: 1,
				},
				credential: projAdmin,
			},
			code: http.StatusBadRequest,
		},

		// 200
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodPut,
				url:    fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				bodyJSON: &models.Label{
					Name:      "product",
					Scope:     common.LabelScopeProject,
					ProjectID: 1,
				},
				credential: projAdmin,
			},
			code: http.StatusOK,
		},
	}

	runCodeCheckingCases(t, cases...)

	label := &models.Label{}
	err := handleAndParse(&testingRequest{
		method: http.MethodGet,
		url:    fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
	}, label)
	require.Nil(t, err)
	assert.Equal(t, "product", label.Name)
}

func TestLabelAPIDelete(t *testing.T) {
	cases := []*codeCheckingCase{
		// 401
		&codeCheckingCase{
			request: &testingRequest{
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
			},
			code: http.StatusUnauthorized,
		},

		// 400
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodDelete,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, 0),
				credential: nonSysAdmin,
			},
			code: http.StatusBadRequest,
		},

		// 404
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodDelete,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, 10000),
				credential: nonSysAdmin,
			},
			code: http.StatusNotFound,
		},

		// 403 non-member user
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodDelete,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				credential: nonSysAdmin,
			},
			code: http.StatusForbidden,
		},

		// 403 developer
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodDelete,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				credential: projDeveloper,
			},
			code: http.StatusForbidden,
		},

		// 200
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodDelete,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				credential: projAdmin,
			},
			code: http.StatusOK,
		},

		// 404
		&codeCheckingCase{
			request: &testingRequest{
				method:     http.MethodDelete,
				url:        fmt.Sprintf("%s/%d", labelAPIBasePath, labelID),
				credential: projAdmin,
			},
			code: http.StatusNotFound,
		},
	}

	runCodeCheckingCases(t, cases...)
}
