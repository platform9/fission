package controller

import (
	"errors"
	"encoding/json"
	"net/http"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
	"strconv"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gorilla/mux"
	"github.com/golang/protobuf/proto"
	"github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/redis/build/gen"
)

func NewClient() redis.Conn {
	// TODO: Load redis ClusterIP from environment variable / configmap
	// TODO: There are two of these functions in different packages -- import?
	c, err := redis.Dial("tcp", "10.102.223.159:6379")
	if err != nil {
		log.Fatalf("Could not connect: %v\n", err)
	}
	return c
}

// TODO: Discuss this approach of using two different protobuf message formats
func deserializeReqResponse(value []byte, reqUID string) (*redisCache.RecordedEntry, error) {
	data := &redisCache.UniqueRequest{}
	err := proto.Unmarshal(value, data)
	if err != nil {
		log.Fatal("Unmarshalling ReqResponse error: ", err)
	}
	log.Info("Parsed protobuf bytes: ", data)
	transformed := &redisCache.RecordedEntry{
		ReqUID: reqUID,
		Req: data.Req,
		Resp: data.Resp,
		Trigger: data.Trigger,
	}
	return transformed, nil
}

func (a *API) RecordsApiListAll(w http.ResponseWriter, r *http.Request) {
	client := NewClient()

	iter := 0
	//var filtered []string
	var filtered []*redisCache.RecordedEntry		// Pointer?

	for {
		arr, err := redis.Values(client.Do("SCAN", iter))
		if err != nil {
			log.Fatal(err)		// TODO: Is this the right thing to do?
		}
		iter, _ = redis.Int(arr[0], nil)
		k, _ := redis.Strings(arr[1], nil)
		for _, key := range k {
			if strings.HasPrefix(key, "REQ") {
				val, err := redis.Bytes(client.Do("HGET", key, "ReqResponse"))
				if err != nil {
					log.Fatal(err)
				}
				entry, err := deserializeReqResponse(val, key)
				if err != nil {
					log.Fatal(err)
				}
				filtered = append(filtered, entry)
			}
		}

		if iter == 0 {
			break
		}
	}

	//log.Info("Printing records: ", filtered)

	resp, err := json.Marshal(filtered)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func validateSplit(timeInput string) (int64, time.Duration, error) {
	num := timeInput[0:len(timeInput)-1]
	unit := string(timeInput[len(timeInput)-1:])

	num2, err := strconv.Atoi(num)
	if err != nil {
		return -1, time.Hour, err		// Return nil time struct?
	}

	num3 := int64(num2)

	log.Info("Parsed time thusly: ", num3, unit, len(unit))

	switch unit {
	case "s":
		return num3, time.Second, nil
	case "m":
		return num3, time.Minute, nil
	case "h":
		return num3, time.Hour, nil
	case "d":
		return num3, 24 * time.Hour, nil
	default:
		log.Info("Failed to default.")
		return -1, time.Hour, errors.New("Invalid time unit")		//TODO: Think of this case
	}
}

// Input: `from` (hours ago, between 0 [today] and 5) and `to` (same units)
// TODO: End range (validate as well)
// Note: Fractional values don't seem to work -- document that for the user
func (a *API) RecordsApiFilterByTime(w http.ResponseWriter, r *http.Request) {
	fromInput := r.FormValue("from")
	toInput := r.FormValue("to")

	// TODO: Reduce duplicate code
	fromMultiplier, fromUnit, err := validateSplit(fromInput)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	toMultiplier, toUnit, err := validateSplit(toInput)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	now := time.Now()
	then := now.Add(time.Duration(-fromMultiplier) * fromUnit)
	rangeStart := then.UnixNano()

	until := now.Add(time.Duration(-toMultiplier) * toUnit)
	rangeEnd := until.UnixNano()

	log.Info("Searching interval, from: ", then, ", to: ", until)

	client := NewClient()

	iter := 0
	var filtered []*redisCache.RecordedEntry

	for {
		arr, err := redis.Values(client.Do("SCAN", iter))
		if err != nil {
			log.Fatal(err)		// TODO: Is this the right thing toInput do?
		}
		iter, _ = redis.Int(arr[0], nil)
		k, _ := redis.Strings(arr[1], nil)
		for _, key := range k {
			if strings.HasPrefix(key, "REQ") {
				val, err := redis.Strings(client.Do("HMGET", key, "Timestamp"))
				if err != nil {
					log.Fatal(err)
					// return err
				}
				tsO, _ := strconv.Atoi(val[0])				// TODO: Get int64 precision fromInput here
				ts := int64(tsO)
				if ts > rangeStart && ts < rangeEnd {
					val2, err := redis.Bytes(client.Do("HGET", key, "ReqResponse"))
					if err != nil {
						log.Fatal(err)
					}
					entry, err := deserializeReqResponse(val2, key)
					if err != nil {
						log.Fatal(err)
					}
					filtered = append(filtered, entry)
				}
			}
		}

		if iter == 0 {
			break
		}
	}

	resp, err := json.Marshal(filtered)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}


func (a *API) RecordsApiFilterByTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trigger := vars["trigger"]

	//trigger := a.extractQueryParamFromRequest(r, "trigger")
	log.Info("In redisApi, got trigger: ", trigger)

	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	// Get all recorders and filter out the ones that aren't attached to the queried trigger
	recorders, err := a.fissionClient.Recorders(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var matchingRecorders []string
	for _, recorder := range recorders.Items {
		if len(recorder.Spec.Triggers) > 0 {
			if includesTrigger(recorder.Spec.Triggers, trigger) {
				matchingRecorders = append(matchingRecorders, recorder.Spec.Name)
			}
		}
	}
	log.Info("Matching recorders: ", matchingRecorders)

	// For each matching recorder, for all its corresponding reqUIDs, if the value's Trigger field == queried trigger,
	// add that reqUID to filtered list

	client := NewClient()

	var filtered []*redisCache.RecordedEntry

	// TODO: Account for old/not-yet-deleted entries in the recorder lists
	// Perhaps create a goroutine for cleaning up these missing values
	for _, key := range matchingRecorders {
		val, err := redis.Strings(client.Do("LRANGE", key, "0", "-1"))   // TODO: Prefix that distinguishes recorder lists
		if err != nil {
			// TODO: Handle deleted recorder? Or is this a non-issue because our list of recorders is up to date?
			a.respondWithError(w, err)
		}
		for _, reqUID := range val {
			val, err := redis.Strings(client.Do("HMGET", reqUID, "Trigger"))  // 1-to-1 reqUID - trigger?
			if err != nil {
				log.Fatal(err)
			}
			if val[0] == trigger {
				// TODO: Reconsider multiple commands
				val, err := redis.Bytes(client.Do("HGET", reqUID, "ReqResponse"))
				if err != nil {
					log.Fatal(err)
				}
				entry, err := deserializeReqResponse(val, reqUID)
				if err != nil {
					log.Fatal(err)
				}
				filtered = append(filtered, entry)
			}
		}
	}

	resp, err := json.Marshal(filtered)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func includesTrigger(triggers []v1.TriggerReference, query string) bool {
	for _, trigger := range triggers {
		if trigger.Name == query {
			return true
		}
	}
	return false
}

func (a *API) RecordsApiFilterByFunction(w http.ResponseWriter, r *http.Request) {
	//query := r.FormValue("query")
	vars := mux.Vars(r)
	query := vars["function"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	recorders, err := a.fissionClient.Recorders(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var matchingRecorders []string
	for _, recorder := range recorders.Items {
		if recorder.Spec.Function == query {
			matchingRecorders = append(matchingRecorders, recorder.Spec.Name)
		}
	}
	log.Info("Matching recorders: ", matchingRecorders)

	client := NewClient()

	//var filteredReqUIDs []string
	var filtered []*redisCache.RecordedEntry

	for _, key := range matchingRecorders {
		val, err := redis.Strings(client.Do("LRANGE", key, "0", "-1"))   // TODO: Prefix that distinguishes recorder lists
		if err != nil {
			a.respondWithError(w, err)
		}
		//filteredReqUIDs = append(filteredReqUIDs, val...)
		for _, reqUID := range val {
			// TODO: Check if it still exists, else clean up this value from the cache
			exists, err := redis.Int(client.Do("EXISTS", reqUID))
			if err != nil {
				continue
			}
			if exists > 0 {
				val, err := redis.Bytes(client.Do("HGET", reqUID, "ReqResponse"))
				if err != nil {
					log.Fatal(err)
				}
				entry, err := deserializeReqResponse(val, reqUID)
				if err != nil {
					log.Fatal(err)
				}
				filtered = append(filtered, entry)
			}
		}
	}

	resp, err := json.Marshal(filtered)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}
