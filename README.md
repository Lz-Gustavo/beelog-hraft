# beelog-hraft
Starting from [Lz-Gustavo/raft-demo #affe623](https://github.com/Lz-Gustavo/raft-demo/commit/affe623b3148ecd73b10f196fc1e9a92b2058cba), *beelog-hraft* implements a minimal key-value store backed by [hashicorp/raft consensus algorithm](https://github.com/hashicorp/raft) used on [beelog](https://github.com/Lz-Gustavo/beelog) evaluations. *beelog* recovery protocol allows an efficient recovery by safely discarding unnecessary entries from the command log, preserving safety and increasing availabitly.

TODO: overview the algorithm, link publications, etc

## Usage
1. Build and run the first replica, informing ```-hjoin``` flag with a port to handle join requests to the cluster. If no ```-port``` and ```-raft``` are set, ":11000" and ":12000" are assumed.
	```bash
	go build
	./beelog-hraft -id node0 -hjoin :13000
	```

2. Wait 2sec for leader election, then launch the other replicas configuring different addresses and ids. Join the leader node by passing its address to ```-join```.
	```bash
	./beelog-hraft -id node1 -port :11001 -raft :12001 -join :13000
	./beelog-hraft -id node2 -port :11002 -raft :12002 -join :13000
	```

3. Check [client/README.md](client/README.md) to launch different workloads.

To run *beelog-hraft* under a distributed environment, simply pass nodes IP addresses when setting ```-raft``` and the leader's IP to ```-join``` flag.

```bash
./beelog-hraft -id node1 -raft 192.168.0.2:12001 -join 192.168.0.1:13000
./beelog-hraft -id node2 -raft 192.168.0.3:12002 -join 192.168.0.1:13000
```
