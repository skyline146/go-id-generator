local multiplier = redis.call("INCR", KEYS[1])
local timestamp = redis.call("GET", KEYS[2])

if not timestamp then
    timestamp = redis.call("time")[1]
    redis.call("SET", KEYS[2], timestamp)
end

if multiplier > tonumber(ARGV[1]) then
    local newTimestamp = redis.call("TIME")[1]
    if newTimestamp == tonumber(timestamp) then
        newTimestamp = newTimestamp + 1
    end

    timestamp = newTimestamp
    multiplier = 1

    redis.call("SET", KEYS[1], 1)
    redis.call("SET", KEYS[2], timestamp)
end

return {multiplier, timestamp}