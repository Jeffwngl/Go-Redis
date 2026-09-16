package main

import (
	"strconv"
	"sync"
	"time"
)

var Handlers = map[string]func([]Value) Value{
	"PING":    ping,
	"SET":     set,
	"GET":     get,
	"HSET":    hset,
	"HGET":    hget,
	"HGETALL": hgetall,
	"TTL":     ttl,
	"EXPIRE":  expire,
}

func ping(args []Value) Value {
	return Value{typ: "string", str: "PONG"}
}

// expiration map
var EXPIREs = make(map[string]time.Time)
var EXPIREsMu sync.RWMutex

func expire(args []Value) Value {
	if len(args) != 2 { // TODO: take care of additional args, e.g. XX
		return Value{
			typ: "error",
			str: "Wrong number of arguments for `expire` command, expected 2",
		}
	}

	key := args[0].bulk
	secondsStr := args[1].bulk

	SETsMu.RLock()

	_, ok := SETs[key] // TODO: fix for HSET and SETs so they reference the same key

	SETsMu.RUnlock()

	if !ok {
		return Value{
			typ: "number",
			num: 0,
		}
	}

	seconds, err := strconv.Atoi(secondsStr)
	if err != nil {
		return Value{
			typ: "error",
			str: "Invalid number",
		}
	}

	EXPIREsMu.Lock()

	EXPIREs[key] = time.Now().Add(time.Duration(seconds) * time.Second)

	EXPIREsMu.Unlock()

	return Value{typ: "number", num: 1}
}

func ttl(args []Value) Value {
	if len(args) != 1 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for `ttl` command, expected 1",
		}
	}

	key := args[0].bulk

	EXPIREsMu.RLock()
	SETsMu.RLock()

	expTime, valid := EXPIREs[key]
	_, exists := SETs[key]

	EXPIREsMu.RUnlock()
	SETsMu.RUnlock()

	if !exists {
		return Value{
			typ: "number",
			num: -2,
		}
	}

	if !valid {
		return Value{
			typ: "number",
			num: -1,
		}
	}

	timeLeft := int(time.Until(expTime).Seconds())

	if timeLeft <= 0 {
		deleteKey(key)
		deleteExpiry(key)

		return Value{
			typ: "number",
			num: -2,
		}
	}

	return Value{typ: "number", num: timeLeft}
}

func deleteKey(key string) {
	delete(SETs, key)
}

func deleteExpiry(key string) {
	delete(EXPIREs, key)
}

// SET map
var SETs = map[string]string{}

// use RWMutex so that server can handle requests concurrently
// so that SETs map is not modified by multiple threads at the same time
var SETsMu = sync.RWMutex{}

// handles: SET key value
/*
args := []Value{
    {typ: "bulk", bulk: "key"},
    {typ: "bulk", bulk: "value"},
}
*/
func set(args []Value) Value {
	if len(args) != 2 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for 'set' command, expected 2",
		}
	}

	key := args[0].bulk
	val := args[1].bulk

	SETsMu.Lock()
	SETs[key] = val
	SETsMu.Unlock()

	return Value{typ: "string", str: "OK"}
}

// handles: GET key
func get(args []Value) Value {
	if len(args) != 1 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for 'get' command, expected 1",
		}
	}

	key := args[0].bulk

	SETsMu.RLock()
	val, ok := SETs[key]
	SETsMu.RUnlock()

	if !ok {
		return Value{typ: "null"}
	}

	return Value{typ: "bulk", bulk: val}
}

var HSETs = map[string]map[string]string{}
var HSETsMu = sync.RWMutex{}

func hset(args []Value) Value {
	if len(args) != 3 {
		return Value{typ: "error", str: "Wrong number of arguments for 'hset' command, expected 3"}
	}

	hash := args[0].bulk
	key := args[1].bulk
	val := args[2].bulk

	HSETsMu.Lock()

	_, ok := HSETs[hash]

	if !ok {
		HSETs[hash] = map[string]string{}
	}

	HSETs[hash][key] = val

	HSETsMu.Unlock()

	return Value{typ: "string", str: "OK"}
}

func hget(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: "error", str: "Wrong number of arguments for 'hget' command, expected 2"}
	}

	hash := args[0].bulk
	key := args[1].bulk

	HSETsMu.RLock()
	val, ok := HSETs[hash][key]
	HSETsMu.RUnlock()

	if !ok {
		return Value{typ: "null"}
	}

	return Value{typ: "bulk", bulk: val}
}

func hgetall(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: "error", str: "Wrong number of arguments for 'hgetall' command, expected 1"}
	}

	res := []Value{}

	hash := args[0].bulk

	for key, val := range HSETs[hash] {
		res = append(
			res,
			Value{typ: "bulk", bulk: key},
			Value{typ: "bulk", bulk: val},
		)
	}

	return Value{typ: "array", array: res}
}
