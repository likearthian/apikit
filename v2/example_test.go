package apikit_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/likearthian/apikit/v2/api"
	httptransport "github.com/likearthian/apikit/v2/transport/http"
)

type greetingRequest struct {
	Name string `query:"name"`
}

type greetingResponse struct {
	Message string `json:"message"`
}

func Example() {
	endpoint := api.Endpoint[greetingRequest, greetingResponse](
		func(_ context.Context, request greetingRequest) (greetingResponse, error) {
			return greetingResponse{Message: "hello, " + request.Name}, nil
		},
	)

	decode := func(_ context.Context, request *http.Request) (greetingRequest, error) {
		var input greetingRequest
		err := httptransport.BindURLQuery(&input, request.URL.Query())
		return input, err
	}

	server, err := httptransport.NewServer(
		endpoint,
		decode,
		httptransport.DefaultJSONResponseEncoder[greetingResponse],
	)
	if err != nil {
		panic(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/greet?name=Ada", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	fmt.Print(response.Body.String())
	// Output:
	// {"request_id":"","message":"success","data":{"message":"hello, Ada"}}
}
