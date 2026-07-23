package http

import (
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"testing"
)

type bindFixture struct {
	Name   string `query:"name"`
	Count  int
	Active bool
	Tags   []string `query:"tag"`
}

var errRejectBinding = errors.New("reject binding")

type rejectingBinding string

func (*rejectingBinding) UnmarshalText([]byte) error {
	return errRejectBinding
}

type nilPanickingTextMarshaler string

func (value *nilPanickingTextMarshaler) MarshalText() ([]byte, error) {
	if value == nil {
		panic("MarshalText called on nil receiver")
	}
	return []byte(*value), nil
}

func TestBindURLQueryConvertsScalarsAndSlices(t *testing.T) {
	query := url.Values{
		"name":   {"Ada"},
		"count":  {"3"},
		"active": {"true"},
		"tag":    {"go,http"},
	}
	var got bindFixture

	if err := BindURLQuery(&got, query); err != nil {
		t.Fatalf("BindURLQuery() error = %v", err)
	}

	want := bindFixture{
		Name:   "Ada",
		Count:  3,
		Active: true,
		Tags:   []string{"go", "http"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BindURLQuery() = %#v, want %#v", got, want)
	}
}

func TestBindURLQueryReportsFieldAndCause(t *testing.T) {
	var got struct {
		Count int `query:"count"`
	}

	err := BindURLQuery(&got, url.Values{"count": {"not-an-int"}})

	var bindingErr *BindingError
	if !errors.As(err, &bindingErr) {
		t.Fatalf("BindURLQuery() error = %v, want *BindingError", err)
	}
	if bindingErr.Field != "count" {
		t.Errorf("BindingError.Field = %q, want %q", bindingErr.Field, "count")
	}
	if bindingErr.Value != "not-an-int" {
		t.Errorf("BindingError.Value = %q, want %q", bindingErr.Value, "not-an-int")
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("BindURLQuery() error = %v, want strconv.ErrSyntax", err)
	}
}

func TestBindURLQueryReportsTextUnmarshalerFieldAndCause(t *testing.T) {
	var got struct {
		Token rejectingBinding `query:"token"`
	}

	err := BindURLQuery(&got, url.Values{"token": {"invalid-token"}})

	var bindingErr *BindingError
	if !errors.As(err, &bindingErr) {
		t.Fatalf("BindURLQuery() error = %v, want *BindingError", err)
	}
	if bindingErr.Field != "token" {
		t.Errorf("BindingError.Field = %q, want %q", bindingErr.Field, "token")
	}
	if bindingErr.Value != "invalid-token" {
		t.Errorf("BindingError.Value = %q, want %q", bindingErr.Value, "invalid-token")
	}
	if !errors.Is(err, errRejectBinding) {
		t.Errorf("BindURLQuery() error = %v, want errRejectBinding", err)
	}
}

func TestBindURLQueryReportsSliceElementFieldAndCause(t *testing.T) {
	var got struct {
		Counts []int `query:"count"`
	}

	err := BindURLQuery(&got, url.Values{"count": {"1,not-an-int,3"}})

	var bindingErr *BindingError
	if !errors.As(err, &bindingErr) {
		t.Fatalf("BindURLQuery() error = %v, want *BindingError", err)
	}
	if bindingErr.Field != "count" {
		t.Errorf("BindingError.Field = %q, want %q", bindingErr.Field, "count")
	}
	if bindingErr.Value != "not-an-int" {
		t.Errorf("BindingError.Value = %q, want %q", bindingErr.Value, "not-an-int")
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("BindURLQuery() error = %v, want strconv.ErrSyntax", err)
	}
}

func TestBindURLQueryReportsAmbiguousCaseInsensitiveKeysDeterministically(t *testing.T) {
	query := url.Values{
		"COUNT": {"1"},
		"Count": {"2"},
	}
	const want = `ambiguous input keys for field "count": COUNT, Count`

	for i := 0; i < 100; i++ {
		var got struct {
			Count int `query:"count"`
		}

		err := BindURLQuery(&got, query)
		if err == nil || err.Error() != want {
			t.Fatalf("BindURLQuery() error = %v, want %q (iteration %d)", err, want, i)
		}
	}
}

func TestBindURLQueryPrefersExactKeyOverCaseInsensitiveMatches(t *testing.T) {
	query := url.Values{
		"count": {"3"},
		"COUNT": {"1"},
		"Count": {"2"},
	}
	var got struct {
		Count int `query:"count"`
	}

	if err := BindURLQuery(&got, query); err != nil {
		t.Fatalf("BindURLQuery() error = %v", err)
	}
	if got.Count != 3 {
		t.Fatalf("BindURLQuery() Count = %d, want 3", got.Count)
	}
}

func TestBindURLQueryUsesSingleCaseInsensitiveMatch(t *testing.T) {
	var got struct {
		Count int `query:"count"`
	}

	if err := BindURLQuery(&got, url.Values{"COUNT": {"3"}}); err != nil {
		t.Fatalf("BindURLQuery() error = %v", err)
	}
	if got.Count != 3 {
		t.Fatalf("BindURLQuery() Count = %d, want 3", got.Count)
	}
}

func TestBindURLQueryRejectsNonPointer(t *testing.T) {
	if err := BindURLQuery(bindFixture{}, url.Values{"name": {"Ada"}}); err == nil {
		t.Fatal("BindURLQuery() error = nil, want non-nil")
	}
}

func TestBindURLQueryInitializesNilStringMap(t *testing.T) {
	var got map[string]string

	if err := BindURLQuery(&got, url.Values{"name": {"Ada"}}); err != nil {
		t.Fatalf("BindURLQuery() error = %v", err)
	}

	if got["name"] != "Ada" {
		t.Fatalf("BindURLQuery() name = %q, want %q", got["name"], "Ada")
	}
}

func TestBindURLQuerySelectsFirstMapValue(t *testing.T) {
	var got map[string]string

	if err := BindURLQuery(&got, url.Values{"name": {"Ada", "Grace"}}); err != nil {
		t.Fatalf("BindURLQuery() error = %v", err)
	}

	if got["name"] != "Ada" {
		t.Fatalf("BindURLQuery() name = %q, want %q", got["name"], "Ada")
	}
}

func TestBindURLQuerySkipsEmptyMapValue(t *testing.T) {
	got := map[string]string{"existing": "value"}

	if err := BindURLQuery(&got, url.Values{"name": {}}); err != nil {
		t.Fatalf("BindURLQuery() error = %v", err)
	}

	if _, exists := got["name"]; exists {
		t.Fatal("BindURLQuery() added name for an empty input slice")
	}
	if got["existing"] != "value" {
		t.Fatalf("BindURLQuery() existing = %q, want %q", got["existing"], "value")
	}
}

func TestBindURLQueryRejectsNonStringMap(t *testing.T) {
	got := map[string]int{}

	if err := BindURLQuery(&got, url.Values{"count": {"3"}}); err == nil {
		t.Fatal("BindURLQuery() error = nil, want non-nil")
	}
}

func TestBindURLQueryRejectsTypedNilDestination(t *testing.T) {
	var target *map[string]string

	if err := BindURLQuery(target, url.Values{"name": {"Ada"}}); err == nil {
		t.Fatal("BindURLQuery() error = nil, want non-nil")
	}
}

func TestEncodeToURLQueryAcceptsStringKeyMap(t *testing.T) {
	got, err := EncodeToURLQuery(map[string]string{"name": "Ada"}, "query")
	if err != nil {
		t.Fatalf("EncodeToURLQuery() error = %v", err)
	}

	if got.Get("name") != "Ada" {
		t.Fatalf("EncodeToURLQuery() name = %q, want %q", got.Get("name"), "Ada")
	}
}

func TestEncodeToURLQuerySkipsNilPointerMapValue(t *testing.T) {
	got, err := EncodeToURLQuery(map[string]*string{"name": nil}, "query")
	if err != nil {
		t.Fatalf("EncodeToURLQuery() error = %v", err)
	}

	if _, exists := got["name"]; exists {
		t.Fatal("EncodeToURLQuery() added name for a nil pointer value")
	}
}

func TestEncodeToURLQueryEncodesSliceMapValue(t *testing.T) {
	got, err := EncodeToURLQuery(map[string][]string{
		"tag": {"go", "http"},
	}, "query")
	if err != nil {
		t.Fatalf("EncodeToURLQuery() error = %v", err)
	}

	if want := []string{"go", "http"}; !reflect.DeepEqual(got["tag"], want) {
		t.Fatalf("EncodeToURLQuery() tag = %#v, want %#v", got["tag"], want)
	}
}

func TestEncodeToURLQuerySkipsInterfaceWrappedNilTextMarshaler(t *testing.T) {
	var token *nilPanickingTextMarshaler
	tests := map[string]interface{}{
		"map": map[string]interface{}{
			"token": token,
		},
		"struct": struct {
			Token interface{} `query:"token"`
		}{
			Token: token,
		},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeToURLQuery(input, "query")
			if err != nil {
				t.Fatalf("EncodeToURLQuery() error = %v", err)
			}
			if _, exists := got["token"]; exists {
				t.Fatal("EncodeToURLQuery() encoded nil TextMarshaler")
			}
		})
	}
}

func TestEncodeToURLQueryRejectsTypedNilStructPointer(t *testing.T) {
	var input *bindFixture

	if _, err := EncodeToURLQuery(input, "query"); err == nil {
		t.Fatal("EncodeToURLQuery() error = nil, want non-nil")
	}
}

func TestEncodeToURLQuerySkipsUnexportedStructField(t *testing.T) {
	input := struct {
		Name   string `query:"name"`
		secret string `query:"secret"`
	}{
		Name:   "Ada",
		secret: "hidden",
	}

	got, err := EncodeToURLQuery(input, "query")
	if err != nil {
		t.Fatalf("EncodeToURLQuery() error = %v", err)
	}
	if got.Get("name") != "Ada" {
		t.Fatalf("EncodeToURLQuery() name = %q, want %q", got.Get("name"), "Ada")
	}
	if _, exists := got["secret"]; exists {
		t.Fatal("EncodeToURLQuery() encoded unexported field")
	}
}

func TestEncodeToURLQueryRejectsNonStringKeyMap(t *testing.T) {
	tests := map[string]map[int]string{
		"empty":     {},
		"populated": {1: "Ada"},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := EncodeToURLQuery(input, "query"); err == nil {
				t.Fatal("EncodeToURLQuery() error = nil, want non-nil")
			}
		})
	}
}

func TestEncodeToURLQueryEncodesInterfaceWrappedScalars(t *testing.T) {
	got, err := EncodeToURLQuery(map[string]any{
		"name":  "Ada",
		"count": 3,
		"empty": nil,
	}, "query")
	if err != nil {
		t.Fatalf("EncodeToURLQuery() error = %v", err)
	}

	if got.Get("name") != "Ada" {
		t.Fatalf("EncodeToURLQuery() name = %q, want %q", got.Get("name"), "Ada")
	}
	if got.Get("count") != "3" {
		t.Fatalf("EncodeToURLQuery() count = %q, want %q", got.Get("count"), "3")
	}
	if _, exists := got["empty"]; exists {
		t.Fatal("EncodeToURLQuery() added empty for a nil interface value")
	}
}
