package http_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	gomock "github.com/golang/mock/gomock"
	mock_logger "github.com/rudderlabs/rudder-server/mocks/utils/logger"
	"github.com/rudderlabs/rudder-server/services/rsources"
	rsources_http "github.com/rudderlabs/rudder-server/services/rsources/http"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	service := rsources.NewMockJobService(mockCtrl)
	handler := rsources_http.NewHandler(service, mock_logger.NewMockLoggerI(mockCtrl))

	tests := []struct {
		name                 string
		jobRunId             string
		endpoint             string
		method               string
		expectedResponseCode int
		serviceReturnError   error
	}{
		{
			name:                 "basic test",
			jobRunId:             "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}", "123"),
			method:               "DELETE",
			expectedResponseCode: 204,
		},
		{
			name:                 "service returns error test",
			jobRunId:             "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}", "123"),
			method:               "DELETE",
			expectedResponseCode: 500,
			serviceReturnError:   fmt.Errorf("something when wrong"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("endpoint tested:", tt.endpoint)
			service.EXPECT().Delete(gomock.Any(), tt.jobRunId).Return(tt.serviceReturnError).Times(1)

			url := fmt.Sprintf("http://localhost:8080%s", tt.endpoint)
			req, err := http.NewRequest(tt.method, url, nil)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)
			_, err = ioutil.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expectedResponseCode, resp.Code, "required error different than expected")
		})
	}
}

func TestGetStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	service := rsources.NewMockJobService(mockCtrl)
	handler := rsources_http.NewHandler(service, mock_logger.NewMockLoggerI(mockCtrl))

	tests := []struct {
		name                 string
		jobID                string
		endpoint             string
		method               string
		expectedResponseCode int
		filter               map[string][]string
		jobStatus            rsources.JobStatus
		getStatusError       error
		respBody             string
	}{
		{
			name:                 "basic test - get status success",
			jobID:                "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}", "123"),
			method:               "GET",
			expectedResponseCode: 200,
			filter: map[string][]string{
				"task_run_id": {"t1", "t2"},
				"source_id":   {"s1"},
			},
			jobStatus: rsources.JobStatus{
				ID: "123",
				TasksStatus: []rsources.TaskStatus{
					{
						ID: "t1",
						SourcesStatus: []rsources.SourceStatus{
							{
								ID:        "s1",
								Completed: false,
								Stats: rsources.Stats{
									In:     1,
									Out:    1,
									Failed: 0,
								},
								DestinationsStatus: []rsources.DestinationStatus{
									{
										ID:        "d1",
										Completed: false,
										Stats: rsources.Stats{
											In:     1,
											Out:    1,
											Failed: 0,
										},
									},
								},
							},
						},
					},
					{
						ID: "t2",
						SourcesStatus: []rsources.SourceStatus{
							{
								ID:        "s1",
								Completed: false,
								Stats: rsources.Stats{
									In:     1,
									Out:    1,
									Failed: 0,
								},
								DestinationsStatus: []rsources.DestinationStatus{
									{
										ID:        "d2",
										Completed: false,
										Stats: rsources.Stats{
											In:     1,
											Out:    1,
											Failed: 0,
										},
									},
								},
							},
						},
					},
				},
			},
			respBody: `{"id":"123","tasks":[{"id":"t1","sources":[{"id":"s1","completed":false,"stats":{"in":1,"out":1,"failed":0},"destinations":[{"id":"d1","completed":false,"stats":{"in":1,"out":1,"failed":0}}]}]},{"id":"t2","sources":[{"id":"s1","completed":false,"stats":{"in":1,"out":1,"failed":0},"destinations":[{"id":"d2","completed":false,"stats":{"in":1,"out":1,"failed":0}}]}]}]}`,
		},
		{
			name:                 "basic test - GetStatus fails with StatusNotFoundError",
			jobID:                "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}", "123"),
			method:               "GET",
			expectedResponseCode: http.StatusNotFound,
			filter: map[string][]string{
				"task_run_id": {"t1", "t2"},
				"source_id":   {"s1"},
			},
			jobStatus:      rsources.JobStatus{},
			getStatusError: rsources.StatusNotFoundError,
			respBody:       statusNotFoundError,
		},
		{
			name:                 "basic test - GetStatus fails with internal server error",
			jobID:                "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}", "123"),
			method:               "GET",
			expectedResponseCode: 500,
			filter: map[string][]string{
				"task_run_id": {"t1", "t2"},
				"source_id":   {"s1"},
			},
			jobStatus:      rsources.JobStatus{},
			getStatusError: errors.New("GetStatusFailed"),
			respBody:       getStatusFailedError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("endpoint tested:", tt.endpoint)

			filterArg := getArgumentFilter(tt.filter)
			service.EXPECT().GetStatus(gomock.Any(), tt.jobID, filterArg).Return(tt.jobStatus, tt.getStatusError).Times(1)

			basicUrl := fmt.Sprintf("http://localhost:8080%s", tt.endpoint)
			url := withFilter(basicUrl, tt.filter)
			req, err := http.NewRequest(tt.method, url, nil)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expectedResponseCode, resp.Code, "actual response code different than expected")
			require.Equal(t, tt.respBody, string(body), "actual response body different than expected")
		})
	}
}

func TestGetFailedRecords(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	service := rsources.NewMockJobService(mockCtrl)
	handler := rsources_http.NewHandler(service, mock_logger.NewMockLoggerI(mockCtrl))

	tests := []struct {
		name                 string
		jobID                string
		endpoint             string
		method               string
		expectedResponseCode int
		filter               map[string][]string
		failedRecords        rsources.FailedRecords
		failedRecordsError   error
		respBody             string
	}{
		{
			name:                 "basic test - GetFailedRecords succeeds",
			jobID:                "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}/failed-records", "123"),
			method:               "GET",
			expectedResponseCode: http.StatusOK,
			filter: map[string][]string{
				"task_run_id": {"t1", "t2"},
				"source_id":   {"s1"},
			},
			failedRecordsError: nil,
			failedRecords: rsources.FailedRecords{
				{
					JobRunID:      "123",
					TaskRunID:     "t1",
					SourceID:      "s1",
					DestinationID: "d1",
					RecordID:      json.RawMessage(`{"id":"record_123"}`),
				},
			},
			respBody: `[{"job_run_id":"123","task_run_id":"t1","source_id":"s1","destination_id":"d1","record_id":{"id":"record_123"}}]`,
		},
		{
			name:                 "get failed records basic test with no failed records",
			jobID:                "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}/failed-records", "123"),
			method:               "GET",
			expectedResponseCode: 200,
			failedRecordsError:   nil,
			filter: map[string][]string{
				"task_run_id": {"t1", "t2"},
				"source_id":   {"s1"},
			},
			failedRecords: rsources.FailedRecords{},
			respBody:      `[]`,
		},
		{
			name:                 "get failed records basic test - GetFailedRecords fails",
			jobID:                "123",
			endpoint:             prepURL("/v1/job-status/{job_run_id}/failed-records", "123"),
			method:               "GET",
			expectedResponseCode: 500,
			filter: map[string][]string{
				"task_run_id": {"t1", "t2"},
				"source_id":   {"s1"},
			},
			failedRecords:      rsources.FailedRecords{},
			failedRecordsError: errors.New("failed to get failed records"),
			respBody:           failedRecordsRespBody,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("endpoint tested:", tt.endpoint)

			filterArg := getArgumentFilter(tt.filter)
			service.EXPECT().GetFailedRecords(gomock.Any(), tt.jobID, filterArg).Return(tt.failedRecords, tt.failedRecordsError).Times(1)

			basicUrl := fmt.Sprintf("http://localhost:8080%s", tt.endpoint)
			url := withFilter(basicUrl, tt.filter)
			req, err := http.NewRequest(tt.method, url, nil)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expectedResponseCode, resp.Code, "actual response code different than expected")
			require.Equal(t, tt.respBody, string(body), "actual response body different than expected")
		})
	}
}

var failedRecordsRespBody string = `failed to get failed records
`

var statusNotFoundError string = `Status not found
`

var getStatusFailedError string = `GetStatusFailed
`

func getArgumentFilter(filter map[string][]string) rsources.JobFilter {
	var filterArg rsources.JobFilter

	if len(filter["task_run_id"]) != 0 {
		tID := filter["task_run_id"]
		filterArg.TaskRunID = tID
	}
	if len(filter["source_id"]) != 0 {
		sID := filter["source_id"]
		filterArg.SourceID = sID
	}

	return filterArg
}

func withFilter(basicUrl string, filters map[string][]string) string {
	if len(filters) == 0 {
		return basicUrl
	}

	newURL := basicUrl + "?"
	for key, values := range filters {
		for _, val := range values {
			newURL = newURL + key + "=" + val + "&"
		}
	}
	return newURL[:len(newURL)-1]
}

func prepURL(url string, params ...string) string {
	re := regexp.MustCompile(`{.*?}`)
	i := 0
	return string(re.ReplaceAllFunc([]byte(url), func(matched []byte) []byte {
		if i >= len(params) {
			panic(fmt.Sprintf("value for %q not provided", matched))
		}
		v := params[i]
		i++
		return []byte(v)
	}))
}
