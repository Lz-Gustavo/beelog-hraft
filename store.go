package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bl "github.com/Lz-Gustavo/beelog"
	"github.com/Lz-Gustavo/beelog/pb"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

// LogStrategy indexes different implemented approaches for command logging, used on
// various evaluation scenarios.
type LogStrategy int8

const (
	// NotLog ...
	NotLog LogStrategy = iota

	// DiskTrad ...
	DiskTrad

	// InmemTrad ...
	InmemTrad

	// BeelogList ...
	BeelogList

	// BeelogArray ...
	BeelogArray

	// BeelogAVL ...
	BeelogAVL

	// BeelogCircBuffer ...
	BeelogCircBuffer

	// BeelogConcTable ...
	BeelogConcTable
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
	logLevel            = "INFO"
	compressValues      = false

	preInitialize = true
	numInitKeys   = 1000000
	initValueSize = 1024

	// Used in catastrophic fault models, where crash faults must be recoverable even if
	// all nodes presented in the consensus cluster are down. Always set to false in any
	// other cases, because this strong assumption greatly degradates performance.
	catastrophicFaults = false

	// defaultLogStrategy is overwritten to 'DiskTrad' if '-logfolder' flag is provided.
	defaultLogStrategy = BeelogConcTable

	// beelog configuration, ignored if 'defaultLogStrategy' isnt 'Beelog*'
	beelogTick   = bl.Interval
	beelogInmem  = false
	beelogPeriod = 4000 // ignored if not Interval on 'beelogTick'
)

var (
	initValue = []byte(strings.Repeat("!", initValueSize))
)

// Custom configuration over default for testing
func configRaft() *raft.Config {
	config := raft.DefaultConfig()
	config.SnapshotInterval = 24 * time.Hour
	config.SnapshotThreshold = 2 << 62
	config.LogLevel = logLevel
	return config
}

// Custom config over default set by const definitions
func configBeelog() *bl.LogConfig {
	var alg bl.Reducer

	// just some defaults configuration, a real application would opt for a single
	// structure on the first way.
	switch defaultLogStrategy {
	case BeelogList:
		alg = bl.GreedyLt
	case BeelogArray:
		alg = bl.GreedyArray
	case BeelogAVL:
		alg = bl.IterDFSAvl
	case BeelogCircBuffer:
		alg = bl.IterCircBuff
	case BeelogConcTable:
		alg = bl.IterConcTable

	default:
		log.Fatalln("unsupported log strategy for beelog config.")
	}

	return &bl.LogConfig{
		Alg:     alg,
		Tick:    beelogTick,
		Inmem:   beelogInmem,
		Period:  beelogPeriod,
		KeepAll: defaultLogStrategy == BeelogConcTable,
		Fname:   "/tmp/beelog-" + svrID + ".log", // ignored if inmem
	}
}

// Store is a simple key-value store, where all changes are made via Raft consensus.
type Store struct {
	RaftDir  string
	RaftBind string
	inMem    bool

	m          map[string][]byte
	compress   bool
	gzipBuffer bytes.Buffer

	raft   *raft.Raft
	logger hclog.Logger

	Logging  LogStrategy
	LogFile  *os.File
	LogFname string
	logCount uint32 // atomic

	st       bl.Structure
	inMemLog *[]pb.Command
	mu       sync.Mutex
}

// NewStore returns a new Store :)
func NewStore(ctx context.Context, inMem bool) *Store {
	s := &Store{
		m:        make(map[string][]byte),
		inMem:    inMem,
		compress: compressValues,
		logger: hclog.New(&hclog.LoggerOptions{
			Name:   "store",
			Level:  hclog.LevelFromString(logLevel),
			Output: os.Stderr,
		}),
	}
	err := s.initLogConfig(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	if joinHandlerAddr != "" {
		go s.ListenRaftJoins(ctx, joinHandlerAddr)
	}

	if recovHandlerAddr != "" {
		go s.ListenStateTransfer(ctx, recovHandlerAddr)
	}

	if compressValues {
		s.gzipBuffer.Reset()
		wtr := gzip.NewWriter(&s.gzipBuffer)
		wtr.Write([]byte(initValue))

		if err := wtr.Flush(); err != nil {
			log.Fatalln(err)
		}
		if err := wtr.Close(); err != nil {
			log.Fatalln(err)
		}
		initValue = s.gzipBuffer.Bytes()
	}

	if preInitialize {
		for i := 0; i < numInitKeys; i++ {
			s.m[strconv.Itoa(i)] = initValue
		}
	}
	return s
}

func (s *Store) initLogConfig(ctx context.Context) error {
	s.Logging = defaultLogStrategy
	if *logfolder != "" {
		s.Logging = DiskTrad
	}

	var err error
	switch s.Logging {
	case NotLog: // avoid error
		return nil

	case InmemTrad:
		s.inMemLog = &[]pb.Command{}
		break

	case DiskTrad:
		s.LogFname = *logfolder + "logfile-" + svrID + ".log"
		s.LogFile = createWriteFile(s.LogFname, true)
		break

	case BeelogAVL:
		config := configBeelog()
		s.st, err = bl.NewAVLTreeHTWithConfig(config)
		if err != nil {
			return err
		}
		break

	case BeelogList:
		config := configBeelog()
		s.st, err = bl.NewListHTWithConfig(config)
		if err != nil {
			return err
		}
		break

	case BeelogArray:
		config := configBeelog()
		s.st, err = bl.NewArrayHTWithConfig(config)
		if err != nil {
			return err
		}
		break

	case BeelogCircBuffer:
		config := configBeelog()
		s.st, err = bl.NewCircBuffHTWithConfig(ctx, config, 4000)
		if err != nil {
			return err
		}
		break

	case BeelogConcTable:
		config := configBeelog()
		s.st, err = bl.NewConcTableWithConfig(ctx, config)
		if err != nil {
			return err
		}
		break

	default:
		return fmt.Errorf("unknow log strategy '%v' provided", s.Logging)
	}
	return nil
}

// Propose invokes Raft.Apply to propose a new command following protocol's atomic broadcast
// to the application's FSM. Sends an "OK" repply to inform commitment. This procedure applies
// "Get" requisitions to prevent inconsistent reads (that do not follow total ordering). etcd's
// issue #741 gives a good explanation about this problem.
func (s *Store) Propose(msg []byte, svr *Server, clientIP string) error {
	if s.raft.State() != raft.Leader {
		return nil
	}

	f := s.raft.Apply(msg, raftTimeout)
	err := f.Error()
	if err != nil {
		return err
	}

	switch f.Response().(type) {
	case string:
		response := strings.Split(f.Response().(string), "-")
		udpAddr := strings.Join([]string{clientIP, ":", response[0]}, "")
		clientRepply := strings.Join([]string{"OK: ", response[1], "\n"}, "")
		svr.SendUDP(udpAddr, clientRepply)
		break

	default:
		return fmt.Errorf("Unrecognized data response %q", f.Response())
	}
	return nil
}

// testGet returns the value for the given key, just using in unit tests since it results
// in an inconsistence read operation, not following total ordering.
func (s *Store) testGet(key string) string {
	return string(s.m[key])
}

// StartRaft opens the store. If enableSingle is set, and there are no existing peers,
// then this node becomes the first node, and therefore leader, of the cluster.
// localID should be the server identifier for this node.
func (s *Store) StartRaft(enableSingle bool, localID string, localRaftAddr string) error {
	// Setup Raft configuration.
	config := configRaft()
	config.LocalID = raft.ServerID(localID)

	// Setup Raft communication.
	addr, err := net.ResolveTCPAddr("tcp", localRaftAddr)
	if err != nil {
		return err
	}
	transport, err := raft.NewTCPTransport(localRaftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	// Using just in-memory storage (could use boltDB in the key-value application)
	logStore := raft.NewInmemStore()
	stableStore := raft.NewInmemStore()

	// Create a fake snapshot store
	dir := "checkpoints/" + localID
	snapshots, err := raft.NewFileSnapshotStore(dir, 2, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot store: %s", err)
	}

	// Instantiate the Raft systems.
	ra, err := raft.NewRaft(config, (*fsm)(s), logStore, stableStore, snapshots, transport)
	if err != nil {
		return fmt.Errorf("new raft: %s", err)
	}
	s.raft = ra

	if enableSingle {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		ra.BootstrapCluster(configuration)
	}
	return nil
}

// JoinRaft joins a raft node, identified by nodeID and located at addr
func (s *Store) JoinRaft(nodeID, addr string, voter bool) error {
	s.logger.Debug(fmt.Sprintf("received join request for remote node %s at %s", nodeID, addr))
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		s.logger.Error(fmt.Sprintf("failed to get raft configuration: %v", err))
		return err
	}

	for _, rep := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if rep.ID == raft.ServerID(nodeID) || rep.Address == raft.ServerAddress(addr) {

			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if rep.Address == raft.ServerAddress(addr) && rep.ID == raft.ServerID(nodeID) {
				s.logger.Debug(fmt.Sprintf("node %s at %s already member of cluster, ignoring join request", nodeID, addr))
				return nil
			}

			future := s.raft.RemoveServer(rep.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	if voter {
		f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
		if f.Error() != nil {
			return f.Error()
		}
	} else {
		f := s.raft.AddNonvoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
		if f.Error() != nil {
			return f.Error()
		}
	}

	s.logger.Debug(fmt.Sprintf("node %s at %s joined successfully", nodeID, addr))
	return nil
}

// ListenRaftJoins receives incoming join requests to the raft cluster. Its initialized
// when "-hjoin" flag is specified, and it can be set only in the first node in case you
// have a static/imutable cluster architecture
func (s *Store) ListenRaftJoins(ctx context.Context, addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to bind connection at %s: %s", addr, err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return

		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("accept failed: %s", err.Error())
			}
			request, _ := bufio.NewReader(conn).ReadString('\n')

			data := strings.Split(request, "-")
			if len(data) < 3 {
				log.Fatalf("incorrect join request, got: %s", data)
			}

			data[2] = strings.TrimSuffix(data[2], "\n")
			voter, _ := strconv.ParseBool(data[2])
			err = s.JoinRaft(data[0], data[1], voter)
			if err != nil {
				log.Fatalf("failed to join node at %s: %s", data[1], err.Error())
			}
		}
	}
}

// LogStateRecover ...
func (s *Store) LogStateRecover(p, n uint64, activePipe net.Conn) error {
	if n < p {
		return fmt.Errorf("invalid interval request, 'n' must be >= 'p'")
	}
	var (
		logs  []byte
		nLogs int
		err   error
		trad  bool
		cmds  []pb.Command
	)

	switch s.Logging {
	case NotLog:
		return fmt.Errorf("cannot retrieve application-level log from a non-logged application")

	case BeelogList, BeelogArray, BeelogAVL, BeelogCircBuffer:
		logs, err = s.st.RecovBytes(p, n)
		if err != nil {
			return err
		}
		break

	case BeelogConcTable: // current only one that retrieves entire state since origin
		logs, nLogs, err = s.st.(*bl.ConcTable).RecovEntireLog()
		if err != nil {
			return err
		}
		break

	case DiskTrad:
		fd, err := os.OpenFile(s.LogFname, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		defer fd.Close()

		// TODO: must rethink this entire procedure to threat concurrent calls between
		// fsm.Log() and LogStateRecover(). Its currently UNSAFE due to concurrent 'logCount'
		// increment and 'UnmarshalWithLen' calls, and a mutex acquisition on each cmd
		// logging seems too expensive (benchmark).
		count := atomic.LoadUint32(&s.logCount)

		// EOL flag is not mandatory on 'UnmarshalWithLen', and its later written by
		// 'MarshalLogIntoWriter' procedure
		data, err := bl.UnmarshalLogWithLenFromReader(fd, int(count))
		if err != nil {
			return err
		}
		cmds = bl.RetainLogInterval(&data, p, n)
		trad = true
		break

	case InmemTrad:
		s.mu.Lock()
		cmds = bl.RetainLogInterval(s.inMemLog, p, n)
		s.mu.Unlock()
		trad = true
		break

	default:
		return fmt.Errorf("unknow log strategy '%v' provided", s.Logging)
	}

	if trad {
		buff := bytes.NewBuffer(nil)

		// Informing the correct indexes retrieved from the current state. Still
		// in doubt if logs must be retained or not from traditional approaches.
		// If not, must inform '0' and 's.logCount' as retrived indexes.
		err = bl.MarshalLogIntoWriter(buff, &cmds, p, n)
		if err != nil {
			return err
		}

		logs, err = ioutil.ReadAll(buff)
		if err != nil {
			return err
		}
	}

	// write num of commands if multiple logs
	if nLogs > 0 {
		_, err := fmt.Fprintf(activePipe, "%d\n", nLogs)
		if err != nil {
			return err
		}
	}
	_, err = activePipe.Write(logs)
	return err
}

// ListenStateTransfer ...
func (s *Store) ListenStateTransfer(ctx context.Context, addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to bind connection at '%s', error: %s", addr, err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return

		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("accept failed: %s", err.Error())
			}

			req, _ := bufio.NewReader(conn).ReadString('\n')
			data := strings.Split(req, "-")
			if len(data) != 3 {
				log.Fatalf("incorrect state request, got: %s", data)
			}

			data[2] = strings.TrimSuffix(data[2], "\n")
			firstIndex, _ := strconv.Atoi(data[1])
			lastIndex, _ := strconv.Atoi(data[2])

			err = s.LogStateRecover(uint64(firstIndex), uint64(lastIndex), conn)
			if err != nil {
				log.Fatalf("failed to transfer log to node located at '%s', error: %s", data[0], err.Error())
			}

			err = conn.Close()
			if err != nil {
				log.Fatalf("error encountered on connection close: %s", err.Error())
			}
		}
	}
}

func createWriteFile(filename string, logindex bool, extraFlags ...int) *os.File {
	flags := os.O_CREATE | os.O_TRUNC | os.O_WRONLY | os.O_APPEND
	if catastrophicFaults {
		flags = flags | os.O_SYNC
	}

	for _, f := range extraFlags {
		flags = flags | f
	}

	fd, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		log.Fatalln("could not create file '", filename, "', err:", err.Error())
		return nil
	}

	if logindex {
		// Important for the first command log interpretation, and doesnt compromise
		// throughput reading. Informed log during recov will have the right number
		// of commands due to 'bl.MarshalLogIntoWriter()'.
		_, err = fmt.Fprintf(fd, "%d\n%d\n%d\n", uint64(0), uint64(0), -1)
		if err != nil {
			log.Fatalln("Error during file creation:", err.Error())
			return nil
		}
	}
	return fd
}
