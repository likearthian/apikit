package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	gohttp "github.com/likearthian/go-http"
)

// DecodeRequestFunc extracts a user-domain request object from an HTTP
// request object. It's designed to be used in HTTP servers, for server-side
// endpoints. One straightforward DecodeRequestFunc could be something that
// JSON decodes from the request body to the concrete request type.
type DecodeRequestFunc[T any] func(context.Context, *http.Request) (request T, err error)

// EncodeRequestFunc encodes the passed request object into the HTTP request
// object. It's designed to be used in HTTP clients, for client-side
// endpoints. One straightforward EncodeRequestFunc could be something that JSON
// encodes the object directly to the request body.
type EncodeRequestFunc func(context.Context, *http.Request, interface{}) error

// CreateRequestFunc creates an outgoing HTTP request based on the passed
// request object. It's designed to be used in HTTP clients, for client-side
// endpoints. It's a more powerful version of EncodeRequestFunc, and can be used
// if more fine-grained control of the HTTP request is required.
type CreateRequestFunc func(context.Context, interface{}) (*http.Request, error)

// EncodeResponseFunc encodes the passed response object to the HTTP response
// writer. It's designed to be used in HTTP servers, for server-side
// endpoints. One straightforward EncodeResponseFunc could be something that
// JSON encodes the object directly to the response body.
type EncodeResponseFunc[T any] func(context.Context, http.ResponseWriter, T) error

// DecodeResponseFunc extracts a user-domain response object from an HTTP
// response object. It's designed to be used in HTTP clients, for client-side
// endpoints. One straightforward DecodeResponseFunc could be something that
// JSON decodes from the response body to the concrete response type.
type DecodeResponseFunc func(context.Context, *http.Response) (response interface{}, err error)

func CommonGetRequestDecoder[T any](ctx context.Context, r *http.Request) (T, error) {
	var reqObj T

	query := r.URL.Query()
	params, ok := ctx.Value(ContextKeyURLParams).(map[string]string)
	if ok {
		//include params into query to be parsed
		for k, v := range params {
			query.Add(k, v)
		}
	}

	if err := BindURLQuery(&reqObj, query); err != nil {
		return reqObj, err
	}

	return reqObj, nil
}

func CommonPostRequestDecoder[T any](ctx context.Context, r *http.Request) (T, error) {
	var reqObj T

	query := r.URL.Query()
	params, ok := ctx.Value(ContextKeyURLParams).(map[string]string)
	if ok {
		//include params into query to be parsed
		for k, v := range params {
			query.Add(k, v)
		}
	}

	err := json.NewDecoder(r.Body).Decode(&reqObj)
	if err != nil {
		return reqObj, fmt.Errorf("%w: %s", fmt.Errorf("bad request"), err)
	}

	if err := BindURLQuery(&reqObj, query); err != nil {
		return reqObj, err
	}

	return reqObj, nil
}

func CommonFileUploadDecoder[T any, PT FileUploader[T]](ctx context.Context, r *http.Request) (interface{}, error) {
	var reqObj = PT(new(T))

	if err := r.ParseMultipartForm(1024 * 1024 * 5); err != nil {
		return nil, err
	}

	for key := range r.MultipartForm.File {
		file, header, err := r.FormFile(key)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, file); err != nil {
			return nil, err
		}

		reqObj.AddFile(header.Filename, buf.Bytes(), header.Header.Get("content-type"))
	}

	if err := BindFormData(reqObj, r.MultipartForm.Value); err != nil {
		return nil, err
	}

	query := r.URL.Query()
	params, ok := ctx.Value(ContextKeyURLParams).(map[string]string)
	if ok {
		//include params into query to be parsed
		for k, v := range params {
			query.Add(k, v)
		}
	}

	if err := BindURLQuery(reqObj, query); err != nil {
		return nil, err
	}

	return reqObj, nil
}

func CommonFileUploadStreamDecoder[T any, PT FileStreamUploader[T]](ctx context.Context, r *http.Request) (interface{}, error) {
	maxMemory := int64(5 * 1024 * 1024)
	var reqObj = PT(new(T))

	reader, err := r.MultipartReader()
	if err != nil {
		return nil, err
	}

	formData := url.Values{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		name := part.FormName()
		filename := part.FileName()
		header := part.Header
		var b bytes.Buffer
		if filename == "" {
			// value, store as string in memory
			n, err := io.CopyN(&b, part, maxMemory+1)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if maxMemory-n < 0 {
				return nil, fmt.Errorf("multipart: message to large")
			}
			formData[name] = append(formData[name], b.String())
			continue
		}

		pr, pw := io.Pipe()
		go func(rd io.ReadCloser) {
			defer pw.Close()
			defer rd.Close()
			if _, err := io.Copy(pw, rd); err != nil {
				fmt.Println(err)
			}
		}(part)

		reqObj.AddFileStream(filename, pr, header.Get("content-type"))
		break
	}

	if err := BindFormData(reqObj, formData); err != nil {
		return nil, err
	}

	query := r.URL.Query()
	params, ok := ctx.Value(ContextKeyURLParams).(map[string]string)
	if ok {
		//include params into query to be parsed
		for k, v := range params {
			query.Add(k, v)
		}
	}

	if err := BindURLQuery(reqObj, query); err != nil {
		return nil, err
	}

	return reqObj, nil
}

func MakeCommonHTTPResponseEncoder(encodeFunc func(context.Context, http.ResponseWriter, any) error) httptransport.EncodeResponseFunc {
	return func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
		// res, ok := response.(T)
		// if !ok {
		// 	return fmt.Errorf("failed to encode response. expected %T, got %T", res, response)
		// }

		return encodeFunc(ctx, w, response)
	}
}

func CommonJSONResponseEncoder(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set(gohttp.HeaderContentType, gohttp.HttpContentTypeJson)
	var gw io.Writer = w
	if needGzipped(ctx) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gw = gz
	}

	return json.NewEncoder(gw).Encode(response)
}

func CommonFileResponseEncoder(ctx context.Context, w http.ResponseWriter, response any) error {
	fileres, ok := response.(*FileResponse)
	if !ok {
		return fmt.Errorf("response object is not of type *FileResponse")
	}

	w.Header().Set(gohttp.HeaderContentType, fileres.ContentType)
	w.Header().Set(gohttp.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", fileres.Filename))
	w.WriteHeader(200)

	if _, err := io.Copy(w, fileres.Content); err != nil {
		fileres.Content.Close()
		return err
	}

	return nil
}

type requestDecoderOption struct {
	acceptedFields  map[string]struct{}
	urlParamsGetter func(context.Context) map[string]string
}

func getAcceptFromContext(ctx context.Context) string {
	val := ctx.Value(ContextKeyRequestAccept)
	enc, ok := val.(string)
	if ok {
		encodings := strings.Split(strings.ToLower(enc), ",")
		return strings.TrimSpace(encodings[0])
	}

	return ""
}

func needGzipped(ctx context.Context) bool {
	val := ctx.Value(ContextKeyRequestAcceptEncoding)
	enc, ok := val.(string)
	var gzipped = false
	if ok {
		encodings := strings.Split(strings.ToLower(enc), ",")
		for _, e := range encodings {
			if strings.TrimSpace(e) == "gzip" {
				gzipped = true
			}
		}
	}

	return gzipped
}
