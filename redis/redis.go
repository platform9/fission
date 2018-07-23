package redis

import (
	"net/http"

	"github.com/fission/fission/redis/build/gen"
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"strings"
	"net/url"
	"encoding/json"
	"os"
	"fmt"
)

func NewClient() redis.Conn {
	rd := os.Getenv("REDIS_PORT_6379_TCP_ADDR")			// TODO: Do this here or somewhere earlier?
	rdport := os.Getenv("REDIS_PORT_6379_TCP_PORT")
	redisUrl := fmt.Sprintf("%s:%s", rd, rdport)

	if len(redisUrl) == 0 {
		log.Fatal("Could not reach Redis in cluster at IP ", redisUrl)
	}

	c, err := redis.Dial("tcp", redisUrl)
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v\n", err)
	}
	return c
}

func EndRecord(triggerName string, recorderName string, reqUID string, request *http.Request, originalUrl url.URL, payload string, response *http.Response, namespace string, timestamp int64) {
	// Case where the function should not have been recorded
	if len(reqUID) == 0 {
		return
	}
	//replayed := request.Header.Get("X-Fission-Replayed")

	log.Info("EndRecord: URL > ", originalUrl.String(), " with body: ", payload)

	//if replayed == "true" {
	//	log.Info("This was a replayed request.")
	//	return
	//}

	fullPath := originalUrl.String()

	escPayload := string(json.RawMessage(payload))

	log.Info("Escaped payload parsed: ", escPayload, " and FullPath > ", fullPath)

	client := NewClient()

	url := make(map[string]string)
	url["Host"] = request.URL.Host
	url["Path"] = fullPath // Previously originalUrl.String()	// Previously request.URL.Path
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
		Host:     request.Host,		// Proxied host?
		Form:     form,
		PostForm: postForm,
	}

	resp := &redisCache.Response{
		Status: response.Status,
		StatusCode: int32(response.StatusCode),
	}

	ureq := &redisCache.UniqueRequest {
		Req: req,
		Resp: resp,
		Trigger: triggerName,			// TODO: Why is this here when Trigger is set as a separate field?
	}

	log.Info("Trying to record this UniqueRequest: ")
	log.Info(ureq.String())

	data, err := proto.Marshal(ureq)
	if err != nil {
		log.Fatal("Marshalling UniqueRequest error: ", err)
	}

	_, err = client.Do("HMSET", reqUID, "ReqResponse", data, "Timestamp", timestamp, "Trigger", triggerName)
	if err != nil {
		panic(err)
	}

	_, err = client.Do("LPUSH", recorderName, reqUID)
	if err != nil {
		panic(err)
	}
}
