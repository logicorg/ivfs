package health

import (
	"errors"
	"net/http"
	"testing"

	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

// TestGet checks that the health endpoint reports ok when Temporal answers
// its health check, and a 503 problem when it does not.
func TestGet(t *testing.T) {
	tests := map[string]struct {
		checkErr   error
		wantStatus int
		wantBody   string
	}{
		"Temporal answers": {
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		"Temporal is unreachable": {
			checkErr:   errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   apitest.Problem(http.StatusServiceUnavailable, "Temporal is unreachable"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &mocks.Client{}
			c.On("CheckHealth", mock.Anything, &client.CheckHealthRequest{}).Return(&client.CheckHealthResponse{}, tc.checkErr)

			response := apitest.Serve(t, Register, c, http.MethodGet, "/api/health", "")
			require.Equal(t, tc.wantStatus, response.Code)
			require.JSONEq(t, tc.wantBody, response.Body.String())
			c.AssertExpectations(t)
		})
	}
}
