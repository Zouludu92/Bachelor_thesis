package initialization

import (
	"bt/harness/global"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path"
	"slices"
)

func versionScan(solverPathMap map[string]string, solverList []string, solverPath string, versionList map[string][]string) map[string][][]string {
	result := make(map[string][][]string)
	var currPathSolver string
	var currPathVersion string

	for _, currSolverName := range solverList {
		currPathSolver = path.Join(solverPath, currSolverName)
		availableVersions, err := os.ReadDir(currPathSolver)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("The resquested solver %s is not present in the solver folder", currSolverName)
				result[currSolverName] = [][]string{}
			} else {
				log.Fatal(err)
			}
		} else {
			if len(versionList[currSolverName]) <= 0 {
				for i := 0; i < len(availableVersions); i++ {
					currPathVersion = path.Join(solverPath, currSolverName, availableVersions[i].Name(), solverPathMap[currSolverName], currSolverName)
					file, err := os.Stat(currPathVersion)
					if err != nil {
						if os.IsNotExist(err) {
							log.Printf("The version %s of solver %s has no executable at the given location", availableVersions[i].Name(), currSolverName)
						}
					} else if !file.IsDir() {
						result[currSolverName] = append(result[currSolverName], []string{availableVersions[i].Name(), currPathVersion})
					} else {
						log.Printf("The version %s of solver %s has no executable at the given location", availableVersions[i].Name(), currSolverName)
					}
				}
			} else {
				for _, currVersion := range versionList[currSolverName] {
					currPathVersion = path.Join(solverPath, currSolverName, currVersion, solverPathMap[currSolverName], currSolverName)
					file, err := os.Stat(currPathVersion)
					if err != nil {
						if os.IsNotExist(err) {
							log.Printf("The version %s of solver %s has no executable at the given location", currVersion, currSolverName)
						}
					} else if !file.IsDir() {
						result[currSolverName] = append(result[currSolverName], []string{currVersion, currPathVersion})
					} else {
						log.Printf("The version %s of solver %s has no executable at the given location", currVersion, currSolverName)
					}
				}
			}

		}
	}
	return result
}

func benchmarkScan(logicList []string, benchmarkPath string) map[string][][]string {
	result := make(map[string][][]string)

	var currLogicPath string
	var currTestPath string
	var nbreTest int
	var testAcc []int

	for _, currLogicName := range logicList {
		currLogicPath = path.Join(benchmarkPath, currLogicName)
		availableTests, err := os.ReadDir(currLogicPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("The resquested logics %s has no given benchmarks", currLogicName)
				result[currLogicName] = [][]string{}
			} else {
				log.Fatal(err)
			}
		} else {
			var filterAvailableTest []fs.DirEntry
			for _, entry := range availableTests {
				if !entry.IsDir() {
					filterAvailableTest = append(filterAvailableTest, entry)
				}
			}
			if len(filterAvailableTest) <= 0 {
				log.Printf("The resquested logics %s has no given benchmarks, or they are in subdirectory", currLogicName)
				result[currLogicName] = [][]string{}
				continue
			} else {

				if len(filterAvailableTest) <= global.PerLogicTests {
					//we have less than requested at disposition, just copy everything
					nbreTest = len(filterAvailableTest)
				} else {
					//we have more than requested at dispo, take only what is asked
					nbreTest = global.PerLogicTests
					//if random, pick the files at random and then go to next logic
					if global.RandomTest {
						for len(testAcc) < global.PerLogicTests {
							currIndex := rand.Int() % len(filterAvailableTest)
							if !slices.Contains(testAcc, currIndex) {
								testAcc = append(testAcc, currIndex)
							}
						}
						for i := 0; i < len(testAcc); i++ {
							currTestPath = path.Join(benchmarkPath, currLogicName, filterAvailableTest[testAcc[i]].Name())
							result[currLogicName] = append(result[currLogicName], []string{filterAvailableTest[testAcc[i]].Name(), currTestPath})
						}
						//reinit the acc for the next run
						testAcc = testAcc[0:0]
						//go to next logic
						continue
					}
					//if not random, then copy the amount needed using nbre test which is the minimum between waht is at dispo and what is asked
					for i := 0; i < nbreTest; i++ {
						currTestPath = path.Join(benchmarkPath, currLogicName, filterAvailableTest[i].Name())
						result[currLogicName] = append(result[currLogicName], []string{filterAvailableTest[i].Name(), currTestPath})
					}
				}
			}
		}
	}
	return result
}
