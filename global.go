package global

import (
	"log"
	"strconv"
	"strings"
)

var TestRound int = 3          //ok
var PerLogicTests int = 3      //ok
var RandomTest bool = false    //ok
var IntergratedTimeOut = false //ok
var TotalTest int = 0
var TotalBinaries int = 0

// 0 for default (default is runtime.NumCPU())
var CoresAllowed int           //ok
var ConcurentExec bool = false //ok
var ResetDB bool = false       //ok
var TimeOut int64 = 90         //ok
var ConfiFilePath string = ""  // /home/parallels/bachlorThesis/v2/configFile.txt
var DbUrl string = ""          // postgresql://postgres:Adrien78@localhost:5432/bt

func InitHarnessGlobal(args []string) {
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "-j"):
			optJ, err := strconv.ParseInt(strings.TrimPrefix(arg, "-j"), 10, 32)
			if err != nil {
				log.Fatal("Number of CPU given is not a valid int value, ensure it has the fomrat -jN, for N = number of CPU")
			}
			ConcurentExec = true
			CoresAllowed = int(optJ)
		case strings.HasPrefix(arg, "-tr"):
			opttr, err := strconv.ParseInt(strings.TrimPrefix(arg, "-tr"), 10, 32)
			if err != nil {
				log.Fatal("Number of repetition per test is not a valid int value, ensure it has the fomrat -trN, for N = number of repetition per test")
			}
			TestRound = int(opttr)
		case strings.HasPrefix(arg, "-pl"):
			optPl, err := strconv.ParseInt(strings.TrimPrefix(arg, "-pl"), 10, 32)
			if err != nil {
				log.Fatal("Number of test per logic is not a valid int value, ensure it has the fomrat -plN, for N = number of test per logic")
			}
			PerLogicTests = int(optPl)
		case strings.HasPrefix(arg, "-t"):
			optt, err := strconv.ParseInt(strings.TrimPrefix(arg, "-t"), 10, 32)
			if err != nil {
				log.Fatal("Timeout value given is not a valid int value, ensure it has the fomrat -tN, for N = Timeout per test in second")
			}
			TimeOut = optt
		case strings.HasPrefix(arg, "-config="):
			// vulnerable
			optconf := strings.TrimPrefix(arg, "-config=")
			ConfiFilePath = optconf
		case strings.HasPrefix(arg, "-db="):
			// vulnerable
			optdb := strings.TrimPrefix(arg, "-db=")
			DbUrl = optdb
		case arg == "-rdb":
			ResetDB = true
		case arg == "-intime":
			IntergratedTimeOut = true
		case arg == "-rtest":
			RandomTest = true

		}
	}
	if ConfiFilePath == "" {
		log.Fatal("You need to specify the location of your configuration file")
	} else if DbUrl == "" {
		log.Fatal("You need to specify the URL of the Database to store the result")
	}
}
