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

	"github.com/likearthian/apikit/api"
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
type EncodeRequestFunc func(context.Context, *http.Request, any) error

// CreateRequestFunc creates an outgoing HTTP request based on the passed
// request object. It's designed to be used in HTTP clients, for client-side
// endpoints. It's a more powerful version of EncodeRequestFunc, and can be used
// if more fine-grained control of the HTTP request is required.
type CreateRequestFunc func(context.Context, any) (*http.Request, error)

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

// DefaultGetByIDStringRequestDecoder is a DecodeRequestFunc that can be used to simply decode request query param "id" into string
func DefaultGetByIDStringRequestDecoder(ctx context.Context, r *http.Request) (string, error) {
	query := r.URL.Query()
	params, ok := ctx.Value(ContextKeyURLParams).(map[string]string)
	if ok {
		//include params into query to be parsed
		for k, v := range params {
			query.Set(k, v)
		}
	}

	return query.Get("id"), nil
}

// DefaultGetRequestDecoder is a DecodeRequestFunc that can be used to decode request query params into the request object T
func DefaultGetRequestDecoder[T any](ctx context.Context, r *http.Request) (T, error) {
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

// DefaultPostRequestDecoder is a DecodeRequestFunc that can be used to decode request query params and parse json body into the request object T
func DefaultPostRequestDecoder[T any](ctx context.Context, r *http.Request) (T, error) {
	var reqObj T

	query := r.URL.Query()
	params, ok := ctx.Value(ContextKeyURLParams).(map[string]string)
	if ok {
		//include params into query to be parsed
		for k, v := range params {
			query.Set(k, v)
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

type FileStreamObject struct {
	Name        string
	FileName    string
	ContentType string
	Reader      *io.PipeReader
}

type FileUploadStreamRequestDTO struct {
	Query    url.Values
	FileChan chan FileStreamObject
	ErrChan  chan error
}

func CommonFileUploadStreamDecoder(ctx context.Context, r *http.Request) (FileUploadStreamRequestDTO, error) {
	fileChan := make(chan FileStreamObject)
	errChan := make(chan error)

	query := r.URL.Query()
	params, ok := ctx.Value(ContextKeyURLParams).(map[string]string)

	if ok {
		//include params into query to be parsed
		for k, v := range params {
			query.Add(k, v)
		}
	}

	reader, err := r.MultipartReader()
	if err != nil {
		return FileUploadStreamRequestDTO{}, err
	}

	go func() {
		defer close(fileChan)
		defer close(errChan)

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			if err != nil {
				errChan <- err
				return
			}

			name := part.FormName()
			filename := part.FileName()
			header := part.Header
			if filename == "" {
				// value, store as string in memory
				continue
			}

			pr, pw := io.Pipe()
			go func(rd io.ReadCloser) {
				defer pw.Close()
				defer rd.Close()
				if _, err := io.Copy(pw, rd); err != nil {
					pw.CloseWithError(err)
				}
			}(part)

			fileChan <- FileStreamObject{
				Name:        name,
				FileName:    filename,
				ContentType: header.Get("content-type"),
				Reader:      pr,
			}
		}
	}()

	return FileUploadStreamRequestDTO{
		Query:    query,
		FileChan: fileChan,
		ErrChan:  errChan,
	}, nil
}

// DefaultSingleFileUploadStreamDecoder is a DecodeRequestFunc that can be used to decode request query params and parse multipart form body
// that contains a single file into the request FileStreamUploader interface object T
func DefaultSingleFileUploadStreamDecoder[T any, PT FileStreamUploader[T]](ctx context.Context, r *http.Request) (PT, error) {
	var reqObj = PT(new(T))

	reader, err := r.MultipartReader()
	if err != nil {
		return reqObj, err
	}

	maxMemory := int64(5 * 1024 * 1024)
	formData := url.Values{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		if err != nil {
			return reqObj, err
		}

		name := part.FormName()
		filename := part.FileName()
		header := part.Header
		var b bytes.Buffer
		if filename == "" {
			// value, store as string in memory
			n, err := io.CopyN(&b, part, maxMemory+1)
			if err != nil && err != io.EOF {
				return reqObj, err
			}
			if maxMemory-n < 0 {
				return reqObj, fmt.Errorf("multipart: message to large")
			}
			formData[name] = append(formData[name], b.String())
			continue
		}

		pr, pw := io.Pipe()
		go func(rd io.ReadCloser) {
			defer pw.Close()
			defer rd.Close()
			if _, err := io.Copy(pw, rd); err != nil {
				pw.CloseWithError(err)
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

type FormStreamUploader[T any] interface {
	SetFileStream(formName string, fileName string, reader io.ReadCloser, contentType string)
	*T
}

func CreateMultipartStreamDecoder[T any, PT FormStreamUploader[T]](maxFileSize int64) DecodeRequestFunc[PT] {
	return func(ctx context.Context, r *http.Request) (PT, error) {
		maxDataMemory := int64(5 * 1024 * 1024)
		var reqObj = PT(new(T))

		reader, err := r.MultipartReader()
		if err != nil {
			return nil, err
		}

		formData := url.Values{}
		var jsonData [][]byte
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

			var b = new(bytes.Buffer)
			if filename == "" {
				// value, store as string in memory
				n, err := io.CopyN(b, part, maxDataMemory+1)
				if err != nil && err != io.EOF {
					return nil, err
				}
				if maxDataMemory-n < 0 {
					return nil, fmt.Errorf("%w. multipart: message too large", api.ErrBadRequest)
				}

				contentType := header.Get(HeaderContentType)
				if contentType == HttpContentTypeJson {
					jsonData = append(jsonData, b.Bytes())
				} else {
					formData[name] = append(formData[name], b.String())
				}
				continue
			}

			n, err := io.CopyN(b, part, maxFileSize+1)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if maxFileSize-n < 0 {
				return nil, fmt.Errorf("%w. multipart: file too large", api.ErrBadRequest)
			}

			reqObj.SetFileStream(name, filename, io.NopCloser(b), header.Get("content-type"))
		}

		if err := BindFormData(reqObj, formData); err != nil {
			return nil, err
		}

		for i, _ := range jsonData {
			if err := json.Unmarshal(jsonData[i], reqObj); err != nil {
				return nil, err
			}
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
}

func OldCommonFileUploadStreamDecoder[T any, PT FileStreamUploader[T]](ctx context.Context, r *http.Request) (interface{}, error) {
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

func MakeCommonHTTPResponseEncoder[T any](encodeFunc func(context.Context, http.ResponseWriter, T) error) EncodeResponseFunc[T] {
	return func(ctx context.Context, w http.ResponseWriter, response T) error {
		// res, ok := response.(T)
		// if !ok {
		// 	return fmt.Errorf("failed to encode response. expected %T, got %T", res, response)
		// }

		return encodeFunc(ctx, w, response)
	}
}

// DefaultJSONResponseEncoder is a EncodeResponseFunc that can be used to encode response object into json.
// your response T will be enclosed in a BaseResponse object in Data field.
func DefaultJSONResponseEncoder[T any](ctx context.Context, w http.ResponseWriter, response T) error {
	w.Header().Set(HeaderContentType, HttpContentTypeJson)
	reqID, _ := ReqIDFromContext(ctx)

	payload := api.SuccessResponse(reqID, response)
	var gw io.Writer = w
	if needGzipped(ctx) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gw = gz
	}

	return json.NewEncoder(gw).Encode(payload)
}

// MakeGenericJSONResponseEncoder is a EncodeResponseFunc generator that can be used to encode response object into json.
// you can pass a responseWrapper function to wrap your response object into another object.
// pass nil to encode the response T as is
func MakeGenericJSONResponseEncoder[T any](responseWrapper func(ctx context.Context, response T) any) EncodeResponseFunc[T] {
	return func(ctx context.Context, w http.ResponseWriter, response T) error {
		w.Header().Set(HeaderContentType, HttpContentTypeJson)

		var payload any = response
		if responseWrapper != nil {
			payload = responseWrapper(ctx, response)
		}

		var gw io.Writer = w
		if needGzipped(ctx) {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			gw = gz
		}

		return json.NewEncoder(gw).Encode(payload)
	}
}

// DefaultPagedJSONResponseEncoder is a EncodeResponseFunc that can be used to encode response object into json.
// it need the response PagedData[T], and will be enclosed in a BaseResponse object in Data field.
func DefaultPagedJSONResponseEncoder[T any](ctx context.Context, w http.ResponseWriter, response api.PagedData[T]) error {
	w.Header().Set(HeaderContentType, HttpContentTypeJson)
	reqID, _ := ReqIDFromContext(ctx)

	payload := api.SuccessResponse(reqID, response.Data, response.Pagination)
	var gw io.Writer = w
	if needGzipped(ctx) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gw = gz
	}

	return json.NewEncoder(gw).Encode(payload)
}

func CommonFileResponseEncoder[T FileResponse](ctx context.Context, w http.ResponseWriter, response *FileResponse) error {
	// fileres, ok := response.(*FileResponse)
	// if !ok {
	// 	return fmt.Errorf("response object is not of type *FileResponse")
	// }

	fileres := response

	w.Header().Set(HeaderContentType, fileres.ContentType)
	w.Header().Set(HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", fileres.Filename))
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
