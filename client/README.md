# beelog-hraft/client
This folder organizes client implementations for the *beelog-hraft* key-value store. Before YCSB, most experiments were run through **seqClient_test** environment launched by external bash scripts. The test env approach (workload generation from test procedures) was first considered due to benchmarking metrics, which later proved to be an unecessary overhead.

Other sub-folders are organized as follows:
* **cmd** is a cmdli client, used only on early stage development. Interprets ad-hoc text messages (*i.e.* ```set-x-10```) from stdin and broadcasts them to replicas.

* **ycsb** implements the database interfaces of [go-ycsb](https://github.com/pingcap/go-ycsb), a Go port of the popular [Yahoo! Cloud Serving Benchmarking](https://github.com/brianfrankcooper/YCSB) tool, for the *beelog-hraft* key-value store.

## Usage
* **workload through test procedures:**

	Execute **genClients.sh** or **run.sh** scripts located at **beelog-hraft/scripts** or manually set a workload of a fixed number of messages by running:
	```
		go test beelog-hraft/client -run TestNumMessagesKvstore -count 1 -clients=5 -req=1000 -key=100000 -data=1 -log=0 -config=/path/to/config.toml
	```
	```-clients``` flag corresponds to the number of concurrent clients, **each of them** launching ```-req``` random requests; ```-key``` represents the number of different possible keys interpret by the key-value application; ```-data``` configures the size of proposed values, where **0: 128B**, **1: 1KB**, **2: 4KB**; ```-log``` sets wheter clients should output latency on a file (1 or 0, *i.e.* **true** or **false**); ```-config``` sets the **.toml** config file location (**./client-config.toml** is set if ommited).

	You can also launch a workload with an execution time limit, where ```-time``` flag corresponds to the desired time limit in seconds:
	```
		go test beelog-hraft/client -run TestClientTimeKvstore -count 1 -clients=5 -time=60 -key=100000 -data=1 -log=0 -config=/path/to/config.toml
	```
	Make sure *beelog-hraft/client* is accessable throught ```$GOPATH```.

* **workload through go-ycsb:**

	**ycsb/client.go** is kept only for reference purposes. You can use and follow [this article](https://medium.com/@siddontang/use-go-ycsb-to-benchmark-different-databases-8850f6edb3a7) to import it on go-ycsb or use my [personal fork](https://github.com/Lz-Gustavo/go-ycsb/tree/kvbeelog) from go-ycsb (run from branch **kvbeelog**). Follow [kvbeelog README file](https://github.com/Lz-Gustavo/go-ycsb/blob/kvbeelog/db/kvbeelog/README.md) to compile it and run with different workloads.
	
	The fork basically duplicates the same client implementation used on test procedures, which is surely not a good practice for programmability (*i.e.* different versions will eventually be observed), but is indeed a convenient one.
