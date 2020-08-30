#!/bin/bash

path=/home/lzgustavo/go/src/go-ycsb
#path=/users/gustavo/go/src/go-ycsb

workloads=("workloada")
numDiffKeys=1000000 # 1kk
numOps=10000000 # 10kk
threadCount=1

if [[ $# -ne 3 ]]; then
	echo "usage: $0 'experimentFolder' 'traditional log(0: false; 1: true)' 'beelog interval'"
	exit 1
fi

#echo "compiling go-ycsb..."
#make -C $path

echo "running..."
for i in ${workloads[*]}; do
	if [[ $2 -eq 0 ]]; then
		$path/bin/go-ycsb run localkv -P $path/workloads/${i} -p threadcount=${threadCount} -p localkv.output=${1} -p recordcount=${numDiffKeys} -p operationcount=${numOps} -p localkv.interval=${3}
	else
		$path/bin/go-ycsb run localkv -P $path/workloads/${i} -p threadcount=${threadCount} -p localkv.output=${1} -p recordcount=${numDiffKeys} -p operationcount=${numOps} -p localkv.logfolder=/tmp/
	fi

	echo "finished running for ${i}..."; echo ""
done