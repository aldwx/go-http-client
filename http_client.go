package httpclient

import (
	"bytes"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

// POST 参数
type RequestParams map[string]interface{}

// URL 参数
type RequestQueries map[string]string

// TokenAPI 获取带 token 的 API 地址
func TokenAPI(api, token string) (string, error) {
	queries := RequestQueries{
		"access_token": token,
	}

	return EncodeURL(api, queries)
}

// EncodeURL add and encode parameters.
func EncodeURL(api string, params RequestQueries) (string, error) {
	urlParse, err := url.Parse(api)
	if err != nil {
		return "", errors.WithStack(err)
	}

	query := urlParse.Query()

	for k, v := range params {
		query.Set(k, v)
	}

	urlParse.RawQuery = query.Encode()

	return urlParse.String(), nil
}

// GetQuery returns url query value
func GetQuery(req *http.Request, key string) string {
	if values, ok := req.URL.Query()[key]; ok && len(values) > 0 {
		return values[0]
	}

	return ""
}

// RandomString random string generator
//
// ln length of return string
func RandomString(ln int) string {
	letters := []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, ln)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}

	return string(b)
}

// PostJSON perform a HTTP/POST request with json body
func PostJSON(url string, params interface{}, response interface{}) error {
	_, body, err := PostJSONWithBody(url, params)
	if err != nil {
		return errors.WithStack(err)
	}

	return jsoniter.Unmarshal(body, response)
}

func GetJSON(url string, response interface{}) error {
	_, body, err := fasthttp.Get(nil, url)
	if err != nil {
		return errors.WithStack(err)
	}

	return jsoniter.Unmarshal(body, response)
}

// PostJSONWithBody return with http body.
func PostJSONWithBody(url string, params interface{}) (int, []byte, error) {
	raw, err := jsoniter.Marshal(params)
	if err != nil {
		return 0, nil, errors.WithStack(err)
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req) // 用完需要释放资源

	// 默认是application/x-www-form-urlencoded
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	req.SetRequestURI(url)

	requestBody := raw
	req.SetBody(requestBody)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp) // 用完需要释放资源
	if err := fasthttp.Do(req, resp); err != nil {
		return 0, nil, errors.WithStack(err)
	}
	return resp.StatusCode(), resp.Body(), err
}

// postJSONWithBody return with http body.
func PostJSONWithBody2(url string, params interface{}) (*http.Response, error) {
	reader := new(bytes.Reader)
	if params != nil {
		raw, err := jsoniter.Marshal(params)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		reader = bytes.NewReader(raw)
	}

	return http.Post(url, "application/json; charset=utf-8", reader)
}

func PostFormByFile(url, field, filename string, response interface{}) error {
	// Add your media file
	file, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	return PostForm(url, field, filename, file, response)
}

func PostForm(url, field, filename string, reader io.Reader, response interface{}) error {
	// Prepare a form that you will submit to that URL.
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	fw, err := w.CreateFormFile(field, filename)
	if err != nil {
		return errors.WithStack(err)
	}

	if _, err = io.Copy(fw, reader); err != nil {
		return errors.WithStack(err)
	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return errors.WithStack(err)
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	return jsoniter.NewDecoder(resp.Body).Decode(response)
}
