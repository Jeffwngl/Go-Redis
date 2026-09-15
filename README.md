# Go-Redis
Redis built in the Go programming language.

## Architecture
- `Value` is the internal representation of the RESP value, from the client, the RESP is converted into an array of `Value` structs which is then used within the Redis server.
- `SETs` is the map which contains the data for Redis, both `GET` and `SET` methods modify this map, `syncRWMutex` is used so that the server can handle requests concurrently and that the map is no modified by multiple threads at the same time.
