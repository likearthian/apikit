package api

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestSuccessResponseJSON(t *testing.T) {
	response := SuccessResponse(
		"req-1",
		struct {
			Name string `json:"name"`
		}{Name: "Ada"},
		PaginationDTO{Page: 2, Total: 9},
	)

	got, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal success response: %v", err)
	}

	want := `{"request_id":"req-1","message":"success","data":{"name":"Ada"},"pagination":{"page":2,"total":9}}`
	if string(got) != want {
		t.Fatalf("unexpected JSON:\n got: %s\nwant: %s", got, want)
	}
}

func TestErrorResponseJSON(t *testing.T) {
	response := ErrorResponse("req-2", errors.New("broken"))

	got, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}

	want := `{"request_id":"req-2","error":"broken","data":null}`
	if string(got) != want {
		t.Fatalf("unexpected JSON:\n got: %s\nwant: %s", got, want)
	}
}

func TestBaseResponseLeavesHTTPStatusToTransport(t *testing.T) {
	response := ErrorResponse("req-3", ErrBadRequest)

	if response.Message != "" {
		t.Fatalf("unexpected message: got %q, want empty", response.Message)
	}
	if response.Error == nil {
		t.Fatal("unexpected nil error")
	}
	if *response.Error != ErrBadRequest.Error() {
		t.Fatalf("unexpected error: got %q, want %q", *response.Error, ErrBadRequest.Error())
	}
}

func TestPagedDataJSONFields(t *testing.T) {
	paged := PagedData[string]{
		Data:       "item",
		Pagination: PaginationDTO{Page: 1, Total: 3},
	}

	got, err := json.Marshal(paged)
	if err != nil {
		t.Fatalf("marshal paged data: %v", err)
	}

	want := `{"data":"item","pagination":{"page":1,"total":3}}`
	if string(got) != want {
		t.Fatalf("unexpected JSON:\n got: %s\nwant: %s", got, want)
	}
}
