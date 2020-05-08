package gqlgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"github.com/pkg/errors"
)

// NewClient only take the first Option if given
func NewClient(endpoint string, opt ...*Option) *Client {
	client := &Client{
		Option: &Option{},
	}
	if len(opt) > 0 && opt[0] != nil {
		client.Option = opt[0]
	}
	if client.HTTPClient == nil {
		client.HTTPClient = http.DefaultClient
	}
	client.Endpoint = endpoint
	return client
}

func (c *Client) Do(ctx context.Context, res interface{}, requests ...Request) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	requestsLen := len(requests)
	if requestsLen == 0 {
		return errors.New("graphql request required")
	}
	var (
		singleReq = requestsLen == 1
		resList   []interface{}
	)
	// batch check
	if !singleReq {
		var ok bool
		resList, ok = res.([]interface{})
		if !ok {
			return errors.New("result should be list")
		}
		if len(resList) != requestsLen {
			return errors.New("results list size should be equal to requests size")
		}
	}

	var (
		httpReqBody    bytes.Buffer
		operationsJson []byte
		contentType    string
		err            error
	)
	if singleReq {
		operationsJson, err = json.Marshal(requests[0])
	} else {
		operationsJson, err = json.Marshal(requests)
	}
	if err != nil {
		return errors.Wrap(err, "json encode graphql request")
	}

	graphqlFiles, err := checkFileUpload(singleReq, requests)
	if err != nil {
		return err
	}

	// when uploading file, use http multipart body, otherwise use json body
	// graphql file upload spec: https://github.com/jaydenseric/graphql-multipart-request-spec
	if len(graphqlFiles) > 0 {
		writer := multipart.NewWriter(&httpReqBody)
		contentType = writer.FormDataContentType()

		if err := writer.WriteField("operations", string(operationsJson)); err != nil {
			return errors.Wrap(err, "write multipart operations field")
		}
		graphqlFilesMap := make(map[int][]string)
		i := 0
		for _, file := range graphqlFiles {
			file.index = i
			graphqlFilesMap[i] = file.paths
			i++
		}
		filesMapJson, err := json.Marshal(graphqlFilesMap)
		if err != nil {
			return errors.Wrap(err, "json marshal graphql upload file map")
		}
		if err := writer.WriteField("map", string(filesMapJson)); err != nil {
			return errors.Wrap(err, "write multipart operations field")
		}

		for _, gqlFile := range graphqlFiles {
			fWriter, err := writer.CreateFormFile(fmt.Sprint(gqlFile.index), gqlFile.file.Name)
			if err != nil {
				return errors.Wrap(err, "multipart writer create from file")
			}
			if _, err := io.Copy(fWriter, gqlFile.file.Reader); err != nil {
				return errors.Wrap(err, "copy file for multipart")
			}
		}

		if err := writer.Close(); err != nil {
			return errors.Wrap(err, "close multipart writer")
		}
	} else {
		contentType = "application/json; charset=utf-8"
		_, err = httpReqBody.Write(operationsJson)
		if err != nil {
			return errors.Wrap(err, "http request body write")
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, &httpReqBody)
	if err != nil {
		return err
	}

	// set http request options and headers
	httpReq.Close = c.CloseBody
	for k, v := range c.Headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Accept", "application/json; charset=utf-8")
	if c.BearerAuth != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.BearerAuth)
	}

	if c.Log != nil {
		c.Log(fmt.Sprintf("%s %s %s, headers: %s, body: %s",
			httpReq.Method,
			httpReq.URL,
			httpReq.Proto,
			httpReq.Header,
			string(operationsJson),
		))
	}
	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	savedBody, _ := ioutil.ReadAll(httpResp.Body)
	respJson := string(savedBody)
	if c.Log != nil {
		c.Log(fmt.Sprintf("%s %s %s <Response %s>, headers: %s, body: %s",
			httpReq.Method,
			httpReq.URL,
			httpResp.Proto,
			httpResp.Status,
			httpResp.Header,
			respJson,
		))
	}
	if !c.NotCheckHTTPStatusCode200 && httpResp.StatusCode != http.StatusOK {
		return &HTTPError{
			Response:  httpResp,
			SavedBody: respJson,
		}
	}

	if singleReq {
		resp := response{
			Data: res,
		}
		if err := json.Unmarshal(savedBody, &resp); err != nil {
			return &JsonError{
				OriginError: err,
				Json:        respJson,
			}
		}
		if len(resp.Errors) > 0 {
			return GraphQLErrors(resp.Errors)
		}
	} else {
		resp := make([]response, requestsLen)
		for k, v := range resList {
			resp[k].Data = v
		}
		if err := json.Unmarshal(savedBody, &resp); err != nil {
			return &JsonError{
				OriginError: err,
				Json:        respJson,
			}
		}
		errs := make([]GraphQLError, 0)
		for _, v := range resp {
			if len(v.Errors) > 0 {
				errs = append(errs, v.Errors...)
			}
		}
		if len(errs) > 0 {
			return GraphQLErrors(errs)
		}
	}

	return nil
}

func checkFileUpload(singleReq bool, requests []Request) (res map[io.Reader]*graphQLFileWithPath, err error) {
	res = make(map[io.Reader]*graphQLFileWithPath)
	for reqIndex, request := range requests {
		for varName, varValue := range request.Variables {
			switch o := varValue.(type) {
			case File:
				path := getPath(singleReq, reqIndex, varName)
				if o.Reader == nil {
					err = errFileReaderIsNil(path)
					return
				}
				if pathMap, ok := res[o.Reader]; ok {
					pathMap.paths = append(pathMap.paths, path)
				} else {
					res[o.Reader] = &graphQLFileWithPath{
						file:  &o,
						paths: []string{path},
					}
				}
			case *File:
				path := getPath(singleReq, reqIndex, varName)
				if o == nil || o.Reader == nil {
					err = errFileReaderIsNil(path)
					return
				}
				if pathMap, ok := res[o.Reader]; ok {
					pathMap.paths = append(pathMap.paths, path)
				} else {
					res[o.Reader] = &graphQLFileWithPath{
						file:  o,
						paths: []string{path},
					}
				}
			case []File:
				for fileIndex, file := range o {
					path := getPath(singleReq, reqIndex, varName, fileIndex)
					if file.Reader == nil {
						err = errFileReaderIsNil(path)
						return
					}
					if pathMap, ok := res[file.Reader]; ok {
						pathMap.paths = append(pathMap.paths, path)
					} else {
						res[file.Reader] = &graphQLFileWithPath{
							file:  &file,
							paths: []string{path},
						}
					}
				}
			case []*File:
				for fileIndex, file := range o {
					path := getPath(singleReq, reqIndex, varName, fileIndex)
					if file == nil || file.Reader == nil {
						err = errFileReaderIsNil(path)
						return
					}
					if pathMap, ok := res[file.Reader]; ok {
						pathMap.paths = append(pathMap.paths, path)
					} else {
						res[file.Reader] = &graphQLFileWithPath{
							file:  file,
							paths: []string{path},
						}
					}
				}
			}
		}
	}
	return res, nil
}

func getPath(singleReq bool, reqIndex int, varName string, fileIndex ...int) (res string) {
	if singleReq {
		res = fmt.Sprintf("variables.%s", varName)
	} else {
		res = fmt.Sprintf("%d.variables.%s", reqIndex, varName)
	}
	if len(fileIndex) > 0 {
		res = fmt.Sprintf("%s.%d", res, fileIndex[0])
	}
	return
}

type response struct {
	Errors []GraphQLError `json:"errors,omitempty"`
	Data   interface{}    `json:"data,omitempty"`
}

type graphQLFileWithPath struct {
	index int
	file  *File
	paths []string
}
