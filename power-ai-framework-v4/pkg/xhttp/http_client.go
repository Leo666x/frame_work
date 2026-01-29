package xhttp

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// HttpRequest struct is a composed http request
type HttpRequest struct {
	RawURL      string
	Method      string
	Headers     http.Header
	QueryParams url.Values
	FormData    url.Values
	File        *File
	Body        []byte
}

// HttpClientConfig contains some configurations for http client
type HttpClientConfig struct {
	Timeout          time.Duration
	SSLEnabled       bool
	TLSConfig        *tls.Config
	Compressed       bool
	HandshakeTimeout time.Duration
	ResponseTimeout  time.Duration
	Verbose          bool
	Proxy            *url.URL
}

// defaultHttpClientConfig default client config.
var defaultHttpClientConfig = &HttpClientConfig{
	Timeout:          50 * time.Second,
	Compressed:       false,
	HandshakeTimeout: 10 * time.Second,
	ResponseTimeout:  10 * time.Second,
}

// HttpClient is used for sending http request.
type HttpClient struct {
	*http.Client
	TLS     *tls.Config
	Request *http.Request
	Config  HttpClientConfig
	Context context.Context
}

// NewHttpClient make a HttpClient instance.
func NewHttpClient() *HttpClient {
	client := &HttpClient{
		Client: &http.Client{
			Timeout: defaultHttpClientConfig.Timeout,
			Transport: &http.Transport{
				TLSHandshakeTimeout:   defaultHttpClientConfig.HandshakeTimeout,
				ResponseHeaderTimeout: defaultHttpClientConfig.ResponseTimeout,
				DisableCompression:    !defaultHttpClientConfig.Compressed,
			},
		},
		Config: *defaultHttpClientConfig,
	}

	return client
}

// NewHttpClientWithConfig make a HttpClient instance with pass config.
func NewHttpClientWithConfig(config *HttpClientConfig) *HttpClient {
	if config == nil {
		config = defaultHttpClientConfig
	}

	client := &HttpClient{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSHandshakeTimeout:   config.HandshakeTimeout,
				ResponseHeaderTimeout: config.ResponseTimeout,
				DisableCompression:    !config.Compressed,
			},
		},
		Config: *config,
	}

	if config.SSLEnabled {
		client.TLS = config.TLSConfig
	}

	if config.Proxy != nil {
		transport := client.Client.Transport.(*http.Transport)
		transport.Proxy = http.ProxyURL(config.Proxy)
	}

	return client
}

// SendRequest send http request.
func (client *HttpClient) SendRequest(request *HttpRequest) (*http.Response, error) {
	err := validateRequest(request)
	if err != nil {
		return nil, err
	}

	rawUrl := request.RawURL

	req, err := http.NewRequest(request.Method, rawUrl, bytes.NewBuffer(request.Body))

	if client.Context != nil {
		req, err = http.NewRequestWithContext(client.Context, request.Method, rawUrl, bytes.NewBuffer(request.Body))
	}

	if err != nil {
		return nil, err
	}

	client.setTLS(rawUrl)
	client.setHeader(req, request.Headers)

	err = client.setQueryParam(req, rawUrl, request.QueryParams)
	if err != nil {
		return nil, err
	}

	if request.FormData != nil {
		if request.File != nil {
			err = client.setFormData(req, request.FormData, setFile(request.File))
		} else {
			err = client.setFormData(req, request.FormData, nil)
		}
	}

	client.Request = req

	resp, err := client.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type HttpRequestResponseFunc func([]byte, error) bool

func (client *HttpClient) SendReqByRespDownload(request *HttpRequest, saveFilePath string) error {

	resp, err := client.SendRequest(request)
	if err != nil {
		return err
	}
	out, err := os.Create(saveFilePath)
	if err != nil {
		return err
	}

	defer out.Close()
	_, err = io.Copy(out, resp.Body)

	return err
}

// SendReqByAsyncRespStream 异步结果返回
func (client *HttpClient) SendReqByAsyncRespStream(request *HttpRequest, handler HttpRequestResponseFunc) {

	go func() {
		resp, err := client.SendRequest(request)
		if err != nil {
			handler(nil, err)
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 4096), 1024*512)
		for scanner.Scan() {

			if !handler(scanner.Bytes(), nil) {
				return
			}
			if err = scanner.Err(); err != nil {
				handler(nil, err)
			}
		}
	}()
}

// SendReqBySyncRespStream 同步结果返回
func (client *HttpClient) SendReqBySyncRespStream(request *HttpRequest, handler HttpRequestResponseFunc) {

	resp, err := client.SendRequest(request)
	if err != nil {
		handler(nil, err)
		return
	}

	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1024*512)
	for scanner.Scan() {

		if !handler(scanner.Bytes(), nil) {
			return
		}
		if err = scanner.Err(); err != nil {
			handler(nil, err)
		}
	}
}

// SendReqByRespStruct 将结果映射成结构体
func (client *HttpClient) SendReqByRespStruct(request *HttpRequest, target any) error {

	resp, err := client.SendRequest(request)
	if err != nil {
		return err
	}

	if resp == nil {
		return errors.New("response is empty")
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

// SendReqByRespString 将结果映射成string
func (client *HttpClient) SendReqByRespString(request *HttpRequest) (string, error) {

	resp, err := client.SendRequest(request)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("response is empty")
	}
	defer resp.Body.Close()
	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(rb), nil
}

// SendReqByRespHttpResponse 返回原始http.Response
func (client *HttpClient) SendReqByRespHttpResponse(request *HttpRequest) (*http.Response, error) {
	return client.SendRequest(request)
}

// setTLS set http client transport TLSClientConfig
func (client *HttpClient) setTLS(rawUrl string) {
	if strings.HasPrefix(rawUrl, "https") {
		if transport, ok := client.Client.Transport.(*http.Transport); ok {
			transport.TLSClientConfig = client.TLS
		}
	}
}

// setHeader set http request header
func (client *HttpClient) setHeader(req *http.Request, headers http.Header) {
	if headers == nil {
		headers = make(http.Header)
	}

	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = []string{"*/*"}
	}
	if _, ok := headers["Accept-Encoding"]; !ok && client.Config.Compressed {
		headers["Accept-Encoding"] = []string{"deflate, gzip"}
	}

	req.Header = headers
}

// setQueryParam set http request query string param
func (client *HttpClient) setQueryParam(req *http.Request, reqUrl string, queryParam url.Values) error {
	if queryParam != nil {
		if !strings.Contains(reqUrl, "?") {
			reqUrl = reqUrl + "?" + queryParam.Encode()
		} else {
			reqUrl = reqUrl + "&" + queryParam.Encode()
		}
		u, err := url.Parse(reqUrl)
		if err != nil {
			return err
		}
		req.URL = u
	}
	return nil
}

// setFormData set http request FormData param
func (client *HttpClient) setFormData(req *http.Request, values url.Values, setFile SetFileFunc) error {
	if setFile != nil {
		err := setFile(req, values)
		if err != nil {
			return err
		}
	} else {
		formData := []byte(values.Encode())
		req.Body = io.NopCloser(bytes.NewReader(formData))
		req.ContentLength = int64(len(formData))
	}
	return nil
}

type SetFileFunc func(req *http.Request, values url.Values) error

// File struct is a combination of file attributes
type File struct {
	Content   []byte
	Path      string
	FieldName string
	FileName  string
}

// setFile set parameters for http request formdata file upload
func setFile(f *File) SetFileFunc {
	return func(req *http.Request, values url.Values) error {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for key, vals := range values {
			for _, val := range vals {
				err := writer.WriteField(key, val)
				if err != nil {
					return err
				}
			}
		}

		if f.Content != nil {
			part, err := writer.CreateFormFile(f.FieldName, f.FileName)
			if err != nil {
				return err
			}
			part.Write(f.Content)
		} else if f.Path != "" {
			file, err := os.Open(f.Path)
			if err != nil {
				return err
			}
			defer file.Close()

			part, err := writer.CreateFormFile(f.FieldName, f.FileName)
			if err != nil {
				return err
			}
			_, err = io.Copy(part, file)
			if err != nil {
				return err
			}
		}

		err := writer.Close()
		if err != nil {
			return err
		}

		req.Body = io.NopCloser(body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.ContentLength = int64(body.Len())

		return nil
	}
}

const methods = "[GET],[POST],[PUT],[DELETE],[PATCH],[HEAD],[CONNECT],[OPTIONS],[TRACE]"

// validateRequest check if a request has url, and valid method.
func validateRequest(req *HttpRequest) error {
	if req.RawURL == "" {
		return errors.New("invalid request url")
	}
	if !strings.Contains(methods, fmt.Sprintf("[%s]", strings.ToUpper(req.Method))) {
		return errors.New("invalid request method")
	}
	return nil
}
