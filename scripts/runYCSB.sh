#!/bin/bash

path=/home/lzgustavo/go/src/go-ycsb

numClients=(1 4 7 10 13 16 19)
workloads=("workloadb")
execTime=10 #seconds
numDiffKeys=1000000

if [[ $# -ne 2 ]] && [[ $# -ne 3 ]]
then
	echo "usage 2 args: $0 'experimentFolder' 'logLatency(0: false; 1: true)'"
	echo "usage 3 args: $0 'experimentFolder' 'logLatency(0: false; 1: true)' 'configFilename'"	
	exit 1
fi

# default config location
config=$path/db/kvbeelog/client-config.toml
if [[ $# -eq 3 ]]; then
	${config}=${3}
fi

#echo "compiling go-ycsb..."
#make -C $path

echo "running..."
for i in ${workloads[*]}; do
	for j in ${numClients[*]}; do

		# TODO: account threadcount on target to avoid early saturation
		if [[ $2 -eq 0 ]]; then
			$path/bin/go-ycsb run kvbeelog -P $path/workloads/${i} -p threadcount=${j} -p recordcount=${numDiffKeys} -p operationcount=$((${j} * ${execTime} * 10000)) -p target=10000 -p kvbeelog.config=${config}
		else
			$path/bin/go-ycsb run kvbeelog -P $path/workloads/${i} -p threadcount=${j} -p recordcount=${numDiffKeys} -p operationcount=$((${j} * ${execTime} * 10000)) -p target=10000 -p kvbeelog.config=${config} -p kvbeelog.output=${1}
		fi

		echo "Finished running experiment for ${i} clients..."

		# waiting for server reasource dealloc...
		sleep 10s
	done
	echo "finished running for ${i}..."; echo ""
done
echo "finished!"
