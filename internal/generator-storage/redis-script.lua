local multiplier = redis.call("INCR", KEYS[1])
local timestamp = redis.call("GET", KEYS[2])

if not timestamp then
    timestamp = redis.call("TIME")[1]
    redis.call("SET", KEYS[2], timestamp)
end

timestamp = tonumber(timestamp)

local newTimestamp = tonumber(redis.call("TIME")[1])
if newTimestamp > timestamp then
    timestamp = newTimestamp
    multiplier = 1

    redis.call("SET", KEYS[1], 1)
end

if multiplier > tonumber(ARGV[1]) then
    while (newTimestamp == timestamp) do
        newTimestamp = tonumber(redis.call("TIME")[1])
    end

    timestamp = newTimestamp
    multiplier = 1

    redis.call("SET", KEYS[1], 1)
end

redis.call("SET", KEYS[2], timestamp)

return {multiplier, timestamp}