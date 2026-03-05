package main

import (
	"bt/harness/global"
	"bt/harness/initialization"
	"bt/harness/test"
	"log"
	"os"
)

//channel of result instance
//channel of running instance
//

//bar *progressbar.ProgressBar,

func main() {
	// urlExample := "postgres://username:password@localhost:5432/database_name"

	args := os.Args[1:]
	global.InitHarnessGlobal(args)

	log.Println("=================== SMT SOLVER BENCHMARK ===================")
	log.Printf("Number of run(s) for each test = %v", global.TestRound)
	log.Printf("Number of test(s) per logic requested = %v", global.PerLogicTests)
	log.Printf("Pick random test(s) in each logic = %v", global.RandomTest)
	log.Printf("Resetting Data bases before starting benchmark = %v", global.ResetDB)
	log.Printf("Timeout per test = %v sec", global.TimeOut)
	log.Printf("Use solver integreted timeout function = %v", global.IntergratedTimeOut)
	log.Printf("Run concurretnly %v", global.ConcurentExec)

	log.Println("Scanning for available benchmarks and solvers")
	solverMap, benchmarkMap, timeOutOPtionMap, supportedLogic := initialization.InitBenchmark()

	var nbreVersionBin int
	var nbreTest int

	log.Println("Solver and benchmarks found as per your request:")
	for currSolver := range solverMap {
		nbreVersionBin = nbreVersionBin + len(solverMap[currSolver])
		global.TotalBinaries = global.TotalBinaries + int(nbreVersionBin)
		for _, currLogics := range supportedLogic[currSolver] {
			nbreTest = nbreTest + len(benchmarkMap[currLogics])
		}

		log.Printf("\tFor solver %s we found/you requested %d version(s) and we found/you requested  %d logic(s) to test it against", currSolver, len(solverMap[currSolver]), len(supportedLogic[currSolver]))
		global.TotalTest = global.TotalTest + (nbreVersionBin * nbreTest * global.TestRound)
		nbreTest = 0
		nbreVersionBin = 0
	}
	log.Printf("Starting benchmarks for a total of %d runs of one single test", global.TotalTest)
	log.Println("=============== STARTING BENCHMARK ===============")
	if global.ConcurentExec {
		test.RunTestConcurently(solverMap, benchmarkMap, timeOutOPtionMap, supportedLogic)
	} else {
		test.RunTest(solverMap, benchmarkMap, timeOutOPtionMap, supportedLogic)
	}
	log.Println("=================== BYE ===================")

}
