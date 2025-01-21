local sobject_key = KEYS[1]
local min = ARGV[1]
local max = ARGV[2]
local limit = ARGV[3]
local val = redis.call("ZRANGEBYSCORE", sobject_key, min, max,"limit", 0, limit)
if(next(val) ~= nil) then
    redis.call('zremrangebyrank', sobject_key, 0, #val - 1)
end
return val