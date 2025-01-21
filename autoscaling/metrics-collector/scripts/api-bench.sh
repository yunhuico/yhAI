#!/bin/bash

# Benchmark metrics-collector API

endpoint="http://localhost:10005/metrics"

all=1000
failed=0
sum_time=0

for ((i=1; i<=$all; i++)); do
	t=$(curl -s -w "%{time_total}\n" -o /dev/null ${endpoint})
	# error check
	if [[ $? -ne 0 ]]; then
		echo "execute curl failed, count $((++failed))"
		continue
	fi
	
	echo $t
	sum_time=`echo "$sum_time+$t" | bc`
done

success=`echo "$all-$failed" | bc`
echo "Success: ${success}"
echo "Total Time: ${sum_time}"

avg_time=`echo "scale=6; ${sum_time}/${success}" | bc -l`
echo "Average: ${avg_time}"
