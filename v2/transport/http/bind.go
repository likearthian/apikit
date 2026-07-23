package http

import "net/url"

// BindURLQuery will unmarshal http request query into a struct or map, pointed by dest.
// dest must be a pointer to struct or map
func BindURLQuery(dest interface{}, query url.Values) error {
	return bindData(dest, query, "query")
}

func BindFormData(dest interface{}, formData url.Values) error {
	return bindData(dest, formData, "form")
}

func EncodeToURLQuery(ptr interface{}, tag string) (url.Values, error) {
	q := url.Values{}
	return q, encodeData(q, ptr, tag)
}
