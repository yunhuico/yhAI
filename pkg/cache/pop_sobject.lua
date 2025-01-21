local sobject_data_key = KEYS[1]
local sobject_id = ARGV[1]
local val = redis.call("HGET", sobject_data_key, sobject_id)
if(val~= false) then
    redis.call("HDEL", sobject_data_key, sobject_id)
end
return val