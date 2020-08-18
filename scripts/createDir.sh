#!/bin/bash

if [[ $# -ne 1 ]]
then
	echo "usage: $0 'parentFolderName'"
	exit 1
fi

local=${1}
folders=("beelog-int-1" "beelog-int-10" "beelog-int-100" "beelog-int-1k" "beelog-int-10k" "notlog" "disktrad")

numClients=(1 4 7 10 13 16 19)
#numClients=(12)
#numClients=(1 3 5 7 9 11 13)

dataSizeOptions=(1) #0: 128B, 1: 1KB, 2: 4KB
#dataSizeOptions=(4k 8k 16k)

echo "creating experiment folders..."
mkdir $local

for i in ${folders[*]}
do
	mkdir $local/${i}
	for j in ${dataSizeOptions[*]}
	do
		mkdir $local/${i}/${j}
		for k in ${numClients[*]}
		do
			mkdir $local/${i}/${j}/${k}
		done
	done
done

echo "finished!"
