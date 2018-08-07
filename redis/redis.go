package redis

import (
	"net/http"

	"encoding/json"
	"fmt"
	"github.com/fission/fission/redis/build/gen"
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"strings"
)

func NewClient() redis.Conn {
	rd := os.Getenv("REDIS_SERVICE_HOST") // TODO: Do this here or somewhere earlier?
	rdport := os.Getenv("REDIS_SERVICE_PORT")
	redisUrl := fmt.Sprintf("%s:%s", rd, rdport)

	if len(redisUrl) == 0 {
		log.Error("Could not reach Redis in cluster at IP ", redisUrl)
		return nil
	}

	c, err := redis.Dial("tcp", redisUrl)
	if err != nil {
		log.Error("Could not connect to Redis: %v\n", err)
		return nil
	}
	return c
}

func Record(triggerName string, recorderName string, reqUID string, request *http.Request, originalUrl url.URL, payload string, response *http.Response, namespace string, timestamp int64) {
	// Case where the function should not have been recorded
	if len(reqUID) == 0 {
		return
	}

	fullPath := originalUrl.String()
	escPayload := string(json.RawMessage(payload))

	client := NewClient()
	if client == nil {
		return
	}

	url := make(map[string]string)
	url["Host"] = request.URL.Host
	url["Path"] = fullPath
	url["Payload"] = escPayload

	header := make(map[string]string)
	for key, value := range request.Header {
		header[key] = strings.Join(value, ",")
	}

	form := make(map[string]string)
	for key, value := range request.Form {
		form[key] = strings.Join(value, ",")
	}

	postForm := make(map[string]string)
	for key, value := range request.PostForm {
		postForm[key] = strings.Join(value, ",")
	}

	req := &redisCache.Request{
		Method:   request.Method,
		URL:      url,
		Header:   header,
		Host:     request.Host, // Proxied host?
		Form:     form,
		PostForm: postForm,
	}

	resp := &redisCache.Response{
		Status:     response.Status,
		StatusCode: int32(response.StatusCode),
	}

	ureq := &redisCache.UniqueRequest{
		Req:     req,
		Resp:    resp,
		Trigger: triggerName,
	}

	data, err := proto.Marshal(ureq)
	if err != nil {
		log.Error("Error marshalling request: ", err)
		return
	}

	_, err = client.Do("HMSET", reqUID, "ReqResponse", data, "Timestamp", timestamp, "Trigger", triggerName)
	if err != nil {
		log.Error("Error saving request: ", err)
		return
	}

	_, err = client.Do("LPUSH", recorderName, reqUID)
	if err != nil {
		log.Error("Error saving recorder-request pair: ", err)
		return
	}
}
