package redis

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"

	"github.com/fission/fission/crd"
	"github.com/fission/fission/redis/build/gen"
)

func RecordsListAll() ([]byte, error) {
	client := NewClient()

	iter := 0
	var filtered []*redisCache.RecordedEntry

	for {
		// Each scan yields only a subset of all keys which is why we keep an iter. When iter == 0,
		// Redis tells us there are no keys left to traverse.
		arr, err := redis.Values(client.Do("SCAN", iter))
		if err != nil {
			return []byte{}, err
		}
		iter, _ = redis.Int(arr[0], nil)
		keys, _ := redis.Strings(arr[1], nil)
		for _, key := range keys {
			if strings.HasPrefix(key, "REQ") {
				val, err := redis.Bytes(client.Do("HGET", key, "ReqResponse"))
				if err != nil {
					log.Fatal(err) // TODO: Fatal log or return err to be handled by caller?
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

	resp, err := json.Marshal(filtered)
	if err != nil {
		return []byte{}, err
	}
	return resp, nil
}

// Input: `from` (hours ago, between 0 [today] and 5) and `to` (same units)
// Note: Fractional values don't seem to work -- document that for the user
func RecordsFilterByTime(from string, to string) ([]byte, error) {
	rangeStart, rangeEnd, err := obtainInterval(from, to)
	log.Debug("Interval inferred: ", rangeStart, rangeEnd)

	if rangeStart >= rangeEnd {
		log.Error("Invalid chronology")
		return []byte{}, err
	}

	client := NewClient()

	iter := 0
	var filtered []*redisCache.RecordedEntry

	for {
		arr, err := redis.Values(client.Do("SCAN", iter))
		if err != nil {
			return []byte{}, err
		}
		iter, _ = redis.Int(arr[0], nil)
		keys, _ := redis.Strings(arr[1], nil)
		for _, key := range keys {
			if strings.HasPrefix(key, "REQ") {
				val, err := redis.Strings(client.Do("HMGET", key, "Timestamp"))
				if err != nil {
					log.Fatal(err)
				}
				tsO, _ := strconv.Atoi(val[0])
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
		return []byte{}, err
	}
	return resp, nil
}

func RecordsFilterByTrigger(queried string, recorders *crd.RecorderList, triggers *crd.HTTPTriggerList) ([]byte, error) {
	matchingRecorders := make(map[string]bool)

	// Implicit triggers:
	// Sometimes triggers are not explicitly attached to recorders but we still want to be able to
	// filter records by those triggers; we do so by identifying the functionReference the queried trigger has
	// and finding recorder(s) for that function

	var correspFunction string
	for _, trigger := range triggers.Items {
		if trigger.Metadata.Name == queried {
			correspFunction = trigger.Spec.FunctionReference.Name
		}
	}

	for _, recorder := range recorders.Items {
		if len(recorder.Spec.Triggers) > 0 {
			if includesTrigger(recorder.Spec.Triggers, queried) {
				matchingRecorders[recorder.Spec.Name] = true
			}
		}
		if recorder.Spec.Function == correspFunction {
			matchingRecorders[recorder.Spec.Name] = true
		}
	}

	log.Info("Matching recorders: ", matchingRecorders)

	client := NewClient()

	var filtered []*redisCache.RecordedEntry

	// TODO: Account for old/not-yet-deleted entries in the recorder lists
	for key := range matchingRecorders {
		val, err := redis.Strings(client.Do("LRANGE", key, "0", "-1")) // TODO: Prefix that distinguishes recorder lists
		if err != nil {
			// TODO: Handle deleted recorder? Or is this a non-issue because our list of recorders is up to date?
			return []byte{}, err
		}
		for _, reqUID := range val {
			val, err := redis.Strings(client.Do("HMGET", reqUID, "Trigger")) // 1-to-1 reqUID - trigger?
			if err != nil {
				log.Fatal(err)
			}
			if val[0] == queried {
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
		return []byte{}, err
	}
	return resp, nil
}

func RecordsFilterByFunction(query string, recorders *crd.RecorderList, triggers *crd.HTTPTriggerList) ([]byte, error) {

	// Implicit functions:
	// Sometimes functions are not explicitly attached to recorders but we still want to be able to
	// filter records by those functions; we do so by identifying all triggers recorders are associated with
	// and checking functionReferences for those triggers.

	triggerMap := make(map[string]crd.HTTPTrigger)
	for _, trigger := range triggers.Items {
		triggerMap[trigger.Metadata.Name] = trigger
	}

	matchingRecorders := make(map[string]bool)

	for _, recorder := range recorders.Items {
		if len(recorder.Spec.Function) > 0 && recorder.Spec.Function == query {
			matchingRecorders[recorder.Spec.Name] = true
		} else {
			for _, trigger := range recorder.Spec.Triggers {
				validTrigger, ok := triggerMap[trigger]
				if ok {
					if validTrigger.Spec.FunctionReference.Name == query {
						matchingRecorders[recorder.Spec.Name] = true
					}
				}
			}
		}
	}

	log.Info("Matching recorders: ", matchingRecorders)

	client := NewClient()

	var filtered []*redisCache.RecordedEntry

	for key := range matchingRecorders {
		val, err := redis.Strings(client.Do("LRANGE", key, "0", "-1"))
		// TODO: Prefix that distinguishes recorder lists
		if err != nil {
			return []byte{}, err
		}

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
		return []byte{}, err
	}
	return resp, nil
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
		ReqUID:  reqUID,
		Req:     data.Req,
		Resp:    data.Resp,
		Trigger: data.Trigger,
	}
	return transformed, nil
}

func obtainInterval(from string, to string) (int64, int64, error) {
	now := time.Now()
	parsedFrom, err := time.ParseDuration(from)
	if err != nil {
		return -1, -1, err
	}

	parsedTo, err := time.ParseDuration(to)
	if err != nil {
		return -1, -1, err
	}

	then := now.Add(-1 * parsedFrom) // Start search interval
	rangeStart := then.UnixNano()

	until := now.Add(-1 * parsedTo) // End search interval
	rangeEnd := until.UnixNano()

	return rangeStart, rangeEnd, nil
}

func includesTrigger(triggers []string, query string) bool {
	for _, trigger := range triggers {
		if trigger == query {
			return true
		}
	}
	return false
}
