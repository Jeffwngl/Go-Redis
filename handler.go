package main

import (
	"strconv"
	"strings"
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

// optional args for expire
var ExpireFlags = map[string]func(string, time.Time) bool{
	"NX":      expireNX,
	"XX":      expireXX,
	"GT":      expireGT,
	"LT":      expireLT,
	"Default": expireDefault,
}

func expire(args []Value) Value {
	length := len(args)

	if length < 2 || length > 3 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for `expire` command, expected 2-3",
		}
	}

	key := args[0].bulk
	secondsStr := args[1].bulk
	flag := "Default"

	if length == 3 {
		flag = strings.ToUpper(args[2].bulk)
	}

	seconds, err := strconv.Atoi(secondsStr)
	if err != nil {
		return Value{
			typ: "error",
			str: "Invalid number",
		}
	}

	command, ok := ExpireFlags[flag]
	if !ok {
		return Value{
			typ: "error",
			str: "Invalid flag",
		}
	}

	SETsMu.RLock()

	_, ok = SETs[key] // TODO: fix for HSET and SETs so they reference the same key

	SETsMu.RUnlock()

	if !ok {
		return Value{
			typ: "number",
			num: 0,
		}
	}

	// TODO: fix race condition between above lock
	// ideally, this whole section is protected by one database level lock
	EXPIREsMu.RLock()
	expTime, valid := EXPIREs[key]
	EXPIREsMu.RUnlock()

	if valid && !time.Now().Before(expTime) {
		deleteKey(key)
		deleteExpiry(key)

		return Value{
			typ: "number",
			num: 0,
		}
	}

	newExpiry := time.Now().Add(time.Duration(seconds) * time.Second)

	if !command(key, newExpiry) {
		return Value{
			typ: "number",
			num: 0,
		}
	}

	//	redis treats <= 0 as immediate expiry
	if seconds <= 0 {
		deleteKey(key)
		deleteExpiry(key)

		return Value{
			typ: "number",
			num: 1,
		}
	}

	EXPIREsMu.Lock()
	EXPIREs[key] = newExpiry
	EXPIREsMu.Unlock()

	return Value{
		typ: "number",
		num: 1,
	}
}

func expireNX(key string, newExpiry time.Time) bool {
	EXPIREsMu.RLock()
	_, hasExpiry := EXPIREs[key]
	EXPIREsMu.RUnlock()

	return !hasExpiry
}

func expireXX(key string, newExpiry time.Time) bool {
	EXPIREsMu.RLock()
	_, hasExpiry := EXPIREs[key]
	EXPIREsMu.RUnlock()

	return hasExpiry
}

func expireGT(key string, newExpiry time.Time) bool {
	EXPIREsMu.RLock()
	currExpiry, hasExpiry := EXPIREs[key]
	EXPIREsMu.RUnlock()

	if !hasExpiry {
		return false
	}

	return newExpiry.After(currExpiry)
}

func expireLT(key string, newExpiry time.Time) bool {
	EXPIREsMu.RLock()
	currExpiry, hasExpiry := EXPIREs[key]
	EXPIREsMu.RUnlock()

	// no expiry is treated as infinity
	// so finite expiry is less than it
	if !hasExpiry {
		return true
	}

	return newExpiry.Before(currExpiry)
}

func expireDefault(key string, newExpiry time.Time) bool {
	return true
}

func ttl(args []Value) Value {
	if len(args) != 1 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for `ttl` command, expected 1",
		}
	}

	key := args[0].bulk
	
	// TODO: fix this race condition
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
