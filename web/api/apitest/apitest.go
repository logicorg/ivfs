// Package apitest helps test the web API's areas: it serves requests through
// one area's endpoints, with the gateway's configuration, guard, and session
// check, signed in as User, against a mocked Temporal client that expects the
// workflow each request runs.
//
// What crosses the mocked client is encoded and decoded as a real client and
// worker would, and every successful response is checked against its schema
// in the OpenAPI document, so a test fails on what would fail on the wire.
package apitest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/web/api/gateway"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

// Host is the Host every test request is sent to, and the only one the
// guard allows.
const Host = "localhost:8080"

// User is the user every test request is signed in as.
var User = uuid.MustParse("0192f3a0-0000-7000-8000-000000000001")

// NewSigner returns a signer with a fixed key, for the sessions SignIn adds.
func NewSigner(t testing.TB) *auth.Signer {
	t.Helper()
	signer, err := auth.NewSigner([][]byte{[]byte("apitest-session-key-of-32-bytes!")})
	require.NoError(t, err)
	return signer
}

// SignIn adds to r a session for User that signer signed.
func SignIn(r *http.Request, signer *auth.Signer) {
	r.AddCookie(signer.Session(User, time.Now()))
}

// A Register adds an area's endpoints to an API, as each area's Register
// does.
type Register func(api huma.API, c client.Client)

// A WorkflowCall is the workflow an endpoint is expected to run: the workflow
// function, the params it receives, and what it returns.
type WorkflowCall struct {
	Workflow any
	// Params is the value the workflow receives, of its parameter's type:
	// what the endpoint passes, encoded and decoded into that type.
	Params any
	// Result is what the workflow returns; the endpoint receives it encoded
	// and decoded into the value it reads the result into.
	Result any
	Err    error
	// WorkflowID is the ID the workflow is started under, after the user's
	// prefix, if the case checks it.
	WorkflowID string
}

// Expect sets c up to run call once, and to fail the test on any other
// workflow. With no call, any workflow fails the test.
func (call *WorkflowCall) Expect(t *testing.T, c *mocks.Client) {
	t.Helper()
	if call == nil {
		return
	}
	// The mock runs inside the handler, so a failure is reported with assert
	// rather than ending the test there.
	run := &mocks.WorkflowRun{}
	run.On("Get", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if call.Result != nil && args.Get(1) != nil {
				assert.NoError(t, roundTrip(call.Result, args.Get(1)), "the workflow's result does not survive the data converter")
			}
		}).
		Return(call.Err)
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, isWorkflow(call.Workflow), mock.Anything).
		Run(func(args mock.Arguments) {
			user, _ := tenancy.UserFrom(args.Get(0).(context.Context))
			assert.Equal(t, User, user, "the user the workflow runs as")
			assert.Equal(t, call.Params, received(t, call.Workflow, args[3:]), "the params the workflow receives")
			id := args.Get(1).(client.StartWorkflowOptions).ID
			assert.True(t, strings.HasPrefix(id, tenancy.WorkflowIDPrefix(User)), "the ID %q is not behind the user's prefix", id)
			if call.WorkflowID != "" {
				assert.Equal(t, tenancy.WorkflowIDPrefix(User)+call.WorkflowID, id, "the ID the workflow is started under")
			}
		}).
		Return(run, nil).Once()
}

// isWorkflow matches the workflow function wf.
func isWorkflow(wf any) any {
	return mock.MatchedBy(func(got any) bool {
		return reflect.ValueOf(got).Pointer() == reflect.ValueOf(wf).Pointer()
	})
}

// received is the value workflow receives as its parameter when it is
// started with args, or nil if it takes none.
func received(t *testing.T, workflow any, args []any) any {
	t.Helper()
	wf := reflect.TypeOf(workflow)
	if wf.NumIn() < 2 {
		assert.Empty(t, args, "the workflow takes no params")
		return nil
	}
	if !assert.Len(t, args, 1, "the workflow takes one param") {
		return nil
	}
	param := reflect.New(wf.In(1))
	if !assert.NoError(t, roundTrip(args[0], param.Interface()), "the params do not survive the data converter") {
		return nil
	}
	return param.Elem().Interface()
}

// roundTrip encodes value with Temporal's default data converter, as a
// client or a worker sends it, and decodes it into valuePtr, as the other
// side receives it.
func roundTrip(value, valuePtr any) error {
	dc := converter.GetDefaultDataConverter()
	payload, err := dc.ToPayload(value)
	if err != nil {
		return err
	}
	return dc.FromPayload(payload, valuePtr)
}

// ChildWorkflowError wraps a structured error the way a facade's error
// reaches the gateway: under the child workflow's wrapper, wrapped again by
// the facade.
func ChildWorkflowError(errType, message string) error {
	structured := temporal.NewApplicationError(message, errType)
	childWrapper := temporal.NewApplicationErrorWithCause("child: "+message, "wrapError", structured)
	return fmt.Errorf("facade: %w", childWrapper)
}

// Serve sends a request the UI could send through register's endpoints, with
// c as their Temporal client, signed in as User, and returns the response.
func Serve(t *testing.T, register Register, c client.Client, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	response, _ := serve(t, register, c, method, path, body)
	return response
}

// serve is Serve, and also returns where the request was routed.
func serve(t *testing.T, register Register, c client.Client, method, path, body string) (*httptest.ResponseRecorder, *route) {
	t.Helper()
	signer := NewSigner(t)
	router := chi.NewMux()
	served := &route{}
	router.Use(gateway.Guard([]string{Host}), auth.Require(signer), func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			served.pattern = chi.RouteContext(r.Context()).RoutePattern()
		})
	})
	served.api = humachi.New(router, gateway.NewConfig())
	register(served.api, c)

	request := httptest.NewRequest(method, "http://"+Host+path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	SignIn(request, signer)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response, served
}

// A route is the API a request was served by, and the path pattern it
// matched.
type route struct {
	api     huma.API
	pattern string
}

// validate checks body, the response with status to a method request, against
// the schema the OpenAPI document gives it.
func (r *route) validate(t *testing.T, method string, status int, body []byte) {
	t.Helper()
	path := r.api.OpenAPI().Paths[r.pattern]
	require.NotNil(t, path, "the document has no path %s", r.pattern)
	operation := map[string]*huma.Operation{
		http.MethodGet: path.Get, http.MethodPost: path.Post, http.MethodPut: path.Put,
		http.MethodPatch: path.Patch, http.MethodDelete: path.Delete,
	}[method]
	require.NotNil(t, operation, "the document has no operation %s %s", method, r.pattern)
	response := operation.Responses[strconv.Itoa(status)]
	require.NotNil(t, response, "the document has no %d response for %s", status, operation.OperationID)
	content := response.Content["application/json"]
	require.NotNil(t, content, "the document has no JSON body for the %d response of %s", status, operation.OperationID)

	var value any
	require.NoError(t, json.Unmarshal(body, &value))
	result := &huma.ValidateResult{}
	huma.Validate(r.api.OpenAPI().Components.Schemas, content.Schema, huma.NewPathBuffer([]byte{}, 0),
		huma.ModeReadFromServer, value, result)
	require.Empty(t, result.Errors, "the %d response of %s does not match its schema", status, operation.OperationID)
}

// A Case is one request to an area's endpoint, and what it should do.
type Case struct {
	Method, Path, Body string
	// Call is the workflow the request runs, if it reaches one.
	Call *WorkflowCall
	// Calls are the workflows a request that runs several runs, in any
	// order.
	Calls      []*WorkflowCall
	WantStatus int
	// WantBody is the JSON the response should hold, if the case checks it.
	WantBody string
}

// RunCases serves each case through register's endpoints, and checks that
// it ran the workflow it expects, and no other, and answered as it should,
// with a successful body that matches its schema.
func RunCases(t *testing.T, register Register, cases map[string]Case) {
	t.Helper()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &mocks.Client{}
			tc.Call.Expect(t, c)
			for _, call := range tc.Calls {
				call.Expect(t, c)
			}

			response, served := serve(t, register, c, tc.Method, tc.Path, tc.Body)
			request := strings.TrimSpace(tc.Method + " " + tc.Path + " " + tc.Body)
			require.Equal(t, tc.WantStatus, response.Code, "%s\n%s", request, response.Body.String())
			if tc.WantBody != "" {
				require.JSONEq(t, tc.WantBody, response.Body.String(), request)
			}
			if response.Code < http.StatusMultipleChoices && response.Body.Len() > 0 {
				served.validate(t, tc.Method, response.Code, response.Body.Bytes())
			}
			c.AssertExpectations(t)
		})
	}
}

// Problem is the JSON body of a problem with status and detail, as the
// gateway answers a workflow's structured error.
func Problem(status int, detail string) string {
	return fmt.Sprintf(`{"title":%q,"status":%d,"detail":%q}`, http.StatusText(status), status, detail)
}
