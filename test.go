package test

import (
	"bt/harness/global"
	"bytes"
	"context"
	"log"
	"math"
	"os/exec"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/schollz/progressbar/v3"
)

type testInstance struct {
	Testname   string
	Output     string
	Correct    bool
	Time       int64
	Solver     string
	Version    string
	Logic      string
	Iteration  int
	PathSolver string
	PathTest   string
	TimeOutOpt string
}

var testInstancePool sync.Pool = sync.Pool{New: func() any { return new(testInstance) }}
var bufferPool sync.Pool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
var regExErr *regexp.Regexp = regexp.MustCompile(`(.*(unsat|sat|unknown).*(unsat|sat|unknown).*)|(.*error.*)`)
var regExTime *regexp.Regexp = regexp.MustCompile(`.*(Time|time).*`)
var regExEnd *regexp.Regexp = regexp.MustCompile(".*Expected result (unsat|sat|unknown) but got")
var regExSat *regexp.Regexp = regexp.MustCompile("unsat|sat|unknown")

func RunTest(solverMap map[string][][]string, bencmarkMap map[string][][]string, timeOutOPtionMap map[string]string, supportedLogic map[string][]string) {
	var currInstance *testInstance
	var rows []*testInstance

	conn, err := pgx.Connect(context.Background(), global.DbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	log.Println("Connected to Data base")

	defer conn.Close(context.Background())

	if global.ResetDB {
		log.Println("Reseting Data base")
		conn.Exec(context.Background(), "TRUNCATE TABLE result")
	}

	bar := progressbar.Default(int64(global.TotalTest))

	//loop over logic
	for currSolver := range solverMap {
		for _, currLogic := range supportedLogic[currSolver] {
			for _, currTest := range bencmarkMap[currLogic] {
				for _, currVerSolver := range solverMap[currSolver] {
					for i := 0; i < global.TestRound; i++ {
						currInstance = testInstancePool.Get().(*testInstance)
						currInstance.PathSolver = currVerSolver[1]
						currInstance.PathTest = currTest[1]
						currInstance.TimeOutOpt = timeOutOPtionMap[currSolver]
						currInstance.Testname = currTest[0]
						currInstance.Solver = currSolver
						currInstance.Version = currVerSolver[0]
						currInstance.Logic = currLogic
						currInstance.Iteration = i
						oneTest(currInstance, regExErr, regExTime, regExEnd, regExSat)
						bar.Add(1)
						rows = append(rows, currInstance)
					}
				}
				_, err := conn.CopyFrom(context.Background(), pgx.Identifier{"result"}, []string{"testname", "output", "correct", "time", "solver", "version", "logic", "iteration"}, pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
					return []any{rows[i].Testname, rows[i].Output, rows[i].Correct, rows[i].Time, rows[i].Solver, rows[i].Version, rows[i].Logic, rows[i].Iteration}, nil
				}))
				if err != nil {
					log.Fatal(err)
				}
				for _, inst := range rows {
					testInstancePool.Put(inst)
				}
				rows = rows[0:0]
			}
		}
	}
}
func DBWriter(pool *pgxpool.Pool, chStore <-chan *testInstance) {
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		log.Fatal("Error while acquiring connection from the database pool!!")
	}
	defer conn.Release()
	var batchArray [25]*testInstance
	var batchSlice = batchArray[0:0]

	for {
		select {
		case instance, ok := <-chStore:
			if !ok {
				_, err := conn.CopyFrom(context.Background(), pgx.Identifier{"result"}, []string{"testname", "output", "correct", "time", "solver", "version", "logic", "iteration"}, pgx.CopyFromSlice(len(batchSlice), func(i int) ([]any, error) {
					return []any{batchSlice[i].Testname, batchSlice[i].Output, batchSlice[i].Correct, batchSlice[i].Time, batchSlice[i].Solver, batchSlice[i].Version, batchSlice[i].Logic, batchSlice[i].Iteration}, nil
				}))
				if err != nil {
					log.Fatal(err)
				}
				for _, inst := range batchSlice {
					testInstancePool.Put(inst)
				}
				return
			} else if len(batchSlice) < cap(batchSlice) {
				batchSlice = append(batchSlice, instance)
			} else {
				_, err := conn.CopyFrom(context.Background(), pgx.Identifier{"result"}, []string{"testname", "output", "correct", "time", "solver", "version", "logic", "iteration"}, pgx.CopyFromSlice(len(batchSlice), func(i int) ([]any, error) {
					return []any{batchSlice[i].Testname, batchSlice[i].Output, batchSlice[i].Correct, batchSlice[i].Time, batchSlice[i].Solver, batchSlice[i].Version, batchSlice[i].Logic, batchSlice[i].Iteration}, nil
				}))
				if err != nil {
					log.Fatal(err)
				}
				for _, inst := range batchSlice {
					testInstancePool.Put(inst)
				}
				batchSlice = batchSlice[0:1]
				batchSlice[0] = instance
			}
		default:
			if len(batchSlice) >= 10 {
				_, err := conn.CopyFrom(context.Background(), pgx.Identifier{"result"}, []string{"testname", "output", "correct", "time", "solver", "version", "logic", "iteration"}, pgx.CopyFromSlice(len(batchSlice), func(i int) ([]any, error) {
					return []any{batchSlice[i].Testname, batchSlice[i].Output, batchSlice[i].Correct, batchSlice[i].Time, batchSlice[i].Solver, batchSlice[i].Version, batchSlice[i].Logic, batchSlice[i].Iteration}, nil
				}))
				if err != nil {
					log.Fatal(err)
				}
				for _, inst := range batchSlice {
					testInstancePool.Put(inst)
				}
				batchSlice = batchSlice[0:0]
			} else {
				time.Sleep(25 * time.Millisecond)
			}
		}

	}
}

func Producer(chProdConsum chan<- *testInstance, listSolverJOb [][]string, bencmarkMap map[string][][]string, timeOutOPtionMap map[string]string, supportedLogic map[string][]string) {
	var currInstance *testInstance
	for _, currSolver := range listSolverJOb {
		for _, currLogic := range supportedLogic[currSolver[0]] {
			for _, currTest := range bencmarkMap[currLogic] {
				for i := 0; i < global.TestRound; i++ {
					currInstance = testInstancePool.Get().(*testInstance)
					currInstance.PathSolver = currSolver[2]
					currInstance.PathTest = currTest[1]
					currInstance.TimeOutOpt = timeOutOPtionMap[currSolver[0]]
					currInstance.Testname = currTest[0]
					currInstance.Solver = currSolver[0]
					currInstance.Version = currSolver[1]
					currInstance.Logic = currLogic
					currInstance.Iteration = i
					chProdConsum <- currInstance
				}
			}
		}
	}
}

func Consumer(chProdConsum <-chan *testInstance, chStore chan<- *testInstance, bar *progressbar.ProgressBar) {
	for {
		instance, ok := <-chProdConsum
		if !ok {
			return
		} else {
			oneTest(instance, regExErr, regExTime, regExEnd, regExSat)
			bar.Add(1)
			chStore <- instance
		}
	}

}

func workSpliter(solverMap map[string][][]string, nbreWorker int) (listSolverJOb [][][]string) {
	pieceSize := global.TotalBinaries / nbreWorker
	rest := global.TotalBinaries - pieceSize
	var currPieceJobs [][]string
	var currPieceSize int = 0

	for solver := range solverMap {
		for _, currVersion := range solverMap[solver] {
			if currPieceSize+1 < pieceSize {
				currPieceJobs = append(currPieceJobs, []string{solver, currVersion[0], currVersion[1]})
				currPieceSize = currPieceSize + 1

			} else if currPieceSize+1 < pieceSize+1 && rest > 0 {
				currPieceJobs = append(currPieceJobs, []string{solver, currVersion[0], currVersion[1]})
				currPieceSize = currPieceSize + 1
				rest = rest - 1
			} else {
				currPieceJobs = append(currPieceJobs, []string{solver, currVersion[0], currVersion[1]})
				listSolverJOb = append(listSolverJOb, currPieceJobs)
				currPieceSize = 0
				currPieceJobs = nil
			}
		}
	}
	return
}

func RunTestConcurently(solverMap map[string][][]string, bencmarkMap map[string][][]string, timeOutOPtionMap map[string]string, supportedLogic map[string][]string) {

	var coresDisp int
	var nbreConsumer int
	var nbreProducer int
	var nbreWriter int
	var chanSize int

	if global.CoresAllowed > 0 {
		runtime.GOMAXPROCS(global.CoresAllowed)
		coresDisp = global.CoresAllowed
	} else {
		coresDisp = runtime.NumCPU()
	}

	pool, err := pgxpool.New(context.Background(), global.DbUrl)

	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	log.Println("Connected to Data base")

	defer pool.Close()

	if global.ResetDB {
		log.Println("Reseting Data base")
		pool.Exec(context.Background(), "TRUNCATE TABLE result")
	}

	if coresDisp <= 2 {
		nbreConsumer = 1
		nbreProducer = 1
		nbreWriter = 1
		chanSize = 25
	} else {
		nbreConsumer = int(math.Floor(float64(coresDisp) * 0.80))
		nbreProducer = nbreConsumer/25 + 1
		nbreWriter = nbreConsumer/25 + 1
		chanSize = 25 * nbreWriter
	}

	log.Printf("Running benchmarks concurentlly with %d goroutine for executing tests (%d goroutine(s) to prepare test, %d goroutine(s) to write to DB) on %d CPU", nbreConsumer, nbreProducer, nbreWriter, coresDisp)

	chProdConsum := make(chan *testInstance, chanSize)
	chStore := make(chan *testInstance, chanSize)

	listSolverJOb := workSpliter(solverMap, nbreProducer)

	bar := progressbar.Default(int64(global.TotalTest))

	var wgProducer sync.WaitGroup
	var wgConsumer sync.WaitGroup
	var wgDBWriter sync.WaitGroup

	var currJob [][]string
	for i := 0; i < nbreProducer; i++ {
		wgProducer.Add(1)
		currJob = listSolverJOb[i]
		go func() {
			defer wgProducer.Done()
			Producer(chProdConsum, currJob, bencmarkMap, timeOutOPtionMap, supportedLogic)
		}()
	}

	for i := 0; i < nbreConsumer; i++ {
		wgConsumer.Add(1)
		go func() {
			defer wgConsumer.Done()
			Consumer(chProdConsum, chStore, bar)
		}()
	}

	for i := 0; i < nbreWriter; i++ {
		wgDBWriter.Add(1)
		go func() {
			defer wgDBWriter.Done()
			DBWriter(pool, chStore)
		}()
	}

	// wait on producer to finish then close prodconsume queue
	wgProducer.Wait()
	close(chProdConsum)
	// wait on consumer to finsih and close chStore queue
	wgConsumer.Wait()
	close(chStore)
	// wait on writer to finish
	wgDBWriter.Wait()
}

func oneTest(currIntance *testInstance, regExErr *regexp.Regexp, regExTime *regexp.Regexp, regExEnd *regexp.Regexp, regExSat *regexp.Regexp) {

	var end time.Time
	var start time.Time
	var duration time.Duration
	var tempStr string
	var runningIter *exec.Cmd

	stdout := bufferPool.Get().(*bytes.Buffer)
	stderr := bufferPool.Get().(*bytes.Buffer)
	stdout.Reset()
	stderr.Reset()
	defer bufferPool.Put(stderr)
	defer bufferPool.Put(stdout)

	if currIntance.TimeOutOpt != "" {
		runningIter = exec.Command(currIntance.PathSolver, currIntance.TimeOutOpt, currIntance.PathTest)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(global.TimeOut*1000000000))
		defer cancel()
		runningIter = exec.CommandContext(ctx, currIntance.PathSolver, currIntance.PathTest)
	}
	runningIter.Stdout = stdout
	runningIter.Stderr = stderr
	start = time.Now()
	err := runningIter.Run()
	end = time.Now()
	duration = end.Sub(start)
	if regExErr.Match(stdout.Bytes()) || regExErr.Match(stderr.Bytes()) {
		//Wrong result
		currIntance.Correct = false
		currIntance.Time = int64(duration)
		// if err but still got output, take it (z3)
		if stdout.Len() > 0 {
			tempStr, _ = stdout.ReadString('\n')
			currIntance.Output = regExSat.FindString(tempStr)
		} else {
			// if err and no output, go get the result in the stderr (cvc5, bitwuzla)
			loc := regExEnd.FindIndex(stderr.Bytes())
			if loc != nil {
				currIntance.Output = string(regExSat.Find(stderr.Bytes()[loc[1]:]))
			} else {
				//if err reported by solver but not due to wrong answer, timeout
				currIntance.Correct = false
				currIntance.Time = global.TimeOut * 1000000000
				currIntance.Output = "timeout"
			}
		}
	} else if regExTime.Match(stdout.Bytes()) || regExTime.Match(stderr.Bytes()) {
		//Timeout
		currIntance.Correct = false
		currIntance.Time = global.TimeOut * 1000000000
		currIntance.Output = "timeout"
	} else if err != nil {
		suberr, ok := err.(*exec.ExitError)
		if !ok {
			//not an errror of the solver
			panic(err)
		} else if suberr.ExitCode() == -1 {
			//Timeout
			currIntance.Correct = false
			currIntance.Time = global.TimeOut * 1000000000
			currIntance.Output = "timeout"
		}
	} else {
		currIntance.Correct = true
		currIntance.Time = int64(duration)
		currIntance.Output = regExSat.FindString(string(stdout.Bytes()))
	}
	return
}
