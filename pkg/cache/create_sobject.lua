local sobject_key = KEYS[1]
local sobject_data_key = KEYS[2]
local sobject_id = ARGV[1]
local sobject_id_score = ARGV[2]
local sobject_data = ARGV[3]
redis.call("zadd", sobject_key, sobject_id_score, sobject_id)
redis.call("hset", sobject_data_key, sobject_id,sobject_data )