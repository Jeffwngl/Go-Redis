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
	"COMMAND": command,
}

/*
TOP LEVEL DATABASE
redis stores expiration deadlines as absolute timestamps
*/
type Database struct {
	data    map[string]Value
	expires map[string]time.Time
	mu      sync.RWMutex
}

var DB Database

func ping(args []Value) Value {
	return Value{typ: "string", str: "PONG"}
}

func command(args []Value) Value {
	return Value{typ: "string", str: "Redis Loaded."}
}

// optional flags for expire
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

	action, ok := ExpireFlags[flag]
	if !ok {
		return Value{
			typ: "error",
			str: "Invalid flag",
		}
	}

	DB.mu.RLock()
	expTime, valid := DB.expires[key]
	_, ok = DB.data[key]
	DB.mu.RUnlock()

	if !ok {
		return Value{
			typ: "number",
			num: 0,
		}
	}

	if valid && time.Now().After(expTime) {
		deleteKey(key)
		deleteExpiry(key)

		return Value{
			typ: "number",
			num: 0,
		}
	}

	newExpiry := time.Now().Add(time.Duration(seconds) * time.Second)

	if !action(key, newExpiry) {
		return Value{
			typ: "number",
			num: 0,
		}
	}

	// redis treats <= 0 as immediate expiry
	if seconds <= 0 {
		deleteKey(key)
		deleteExpiry(key)

		return Value{
			typ: "number",
			num: 1,
		}
	}

	DB.mu.Lock()
	DB.expires[key] = newExpiry
	DB.mu.Unlock()

	return Value{
		typ: "number",
		num: 1,
	}
}

func expireNX(key string, newExpiry time.Time) bool {
	DB.mu.RLock()
	_, hasExpiry := DB.expires[key]
	DB.mu.RUnlock()

	return !hasExpiry
}

func expireXX(key string, newExpiry time.Time) bool {
	DB.mu.RLock()
	_, hasExpiry := DB.expires[key]
	DB.mu.RUnlock()

	return hasExpiry
}

func expireGT(key string, newExpiry time.Time) bool {
	DB.mu.RLock()
	currExpiry, hasExpiry := DB.expires[key]
	DB.mu.RUnlock()

	if !hasExpiry {
		return false
	}

	return newExpiry.After(currExpiry)
}

func expireLT(key string, newExpiry time.Time) bool {
	DB.mu.RLock()
	currExpiry, hasExpiry := DB.expires[key]
	DB.mu.RUnlock()

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

	DB.mu.RLock()
	expTime, valid := DB.expires[key]
	_, exists := DB.data[key]
	DB.mu.RUnlock()

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

	/*
		redis stores expiration metadata separately from the main dictionary,
		expired keys are removed either lazily or actively in background expiration
		cycles, redis 6 improved active expiration by using a radix tree containing
		keys likely to expire soon.
	*/
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

// used only inside a database lock
func deleteKey(key string) {
	delete(DB.data, key)
}

func deleteExpiry(key string) {
	delete(DB.expires, key)
}

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
	newVal := args[1].bulk

	DB.mu.RLock()
	val, ok := DB.data[key]
	_, expireExists := DB.expires[key]
	DB.mu.RUnlock()

	if !ok && val.typ != "string" {
		return Value{
			typ: "error",
			str: "WRONGTYPE, key doesn't exist as string.",
		}
	}

	DB.mu.Lock()
	val = Value{
		typ: "string",
		str: newVal,
	}

	if expireExists {
		deleteExpiry(key)
	}
	DB.mu.Unlock()

	return Value{
		typ: "string",
		str: "OK",
	}
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

	val, ok := getLiveKey(key)

	if !ok {
		return Value{typ: "null"}
	}

	if val.typ != "string" {
		return Value{
			typ: "error",
			str: "WRONGTYPE, hash cannot be used as key.",
		}
	}

	return Value{typ: "bulk", bulk: val.str}
}

func hset(args []Value) Value {
	if len(args) != 3 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for 'hset' command, expected 3",
		}
	}

	hash := args[0].bulk
	key := args[1].bulk
	val := args[2].bulk

	DB.mu.Lock()
	defer DB.mu.Unlock()
	hashVal, ok := DB.data[hash]

	if !ok {
		hashVal = Value{
			typ:  "hash",
			hash: map[string]string{},
		}
	}

	if hashVal.typ != "hash" {
		return Value{
			typ: "error",
			str: "WRONGTYPE, hash cannot be the same as key.",
		}
	}

	hashVal.hash[key] = val

	return Value{
		typ: "string",
		str: "OK",
	}
}

func hget(args []Value) Value {
	if len(args) != 2 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for 'hget' command, expected 2",
		}
	}

	hash := args[0].bulk
	key := args[1].bulk

	hashVal, ok := getLiveKey(hash)

	if !ok {
		return Value{typ: "null"}
	}

	if hashVal.typ != "hash" {
		return Value{
			typ: "error",
			str: "WRONGTYPE, hash cannot be the same as key.",
		}
	}

	val, ok := hashVal.hash[key]

	if !ok {
		return Value{typ: "null"}
	}

	return Value{
		typ:  "bulk",
		bulk: val,
	}
}

func hgetall(args []Value) Value {
	if len(args) != 1 {
		return Value{
			typ: "error",
			str: "Wrong number of arguments for 'hgetall' command, expected 1",
		}
	}

	res := []Value{}

	hash := args[0].bulk

	hashVal, ok := getLiveKey(hash)

	if !ok {
		return Value{typ: "null"}
	}

	if hashVal.typ != "hash" {
		return Value{
			typ: "error",
			str: "WRONGTYPE, hash cannot be the same as key.",
		}
	}

	for key, val := range hashVal.hash {
		res = append(
			res,
			Value{typ: "bulk", bulk: key},
			Value{typ: "bulk", bulk: val},
		)
	}

	return Value{typ: "array", array: res}
}

func getLiveKey(key string) (Value, bool) {
	DB.mu.Lock()
	defer DB.mu.Unlock()

	val, ok := DB.data[key]

	if !ok {
		return Value{typ: "null"}, false
	}

	expTime, ok := DB.expires[key]

	if ok && !time.Now().Before(expTime) {
		deleteKey(key)
		deleteExpiry(key)

		return Value{typ: "null"}, false
	}

	return val, true
}
