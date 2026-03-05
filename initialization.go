package initialization

import (
	"bt/harness/global"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func compileTimeoutOpt(regExScale *regexp.Regexp, regExOg *regexp.Regexp, regExNbre *regexp.Regexp, input string) string {
	input = input[1 : len(input)-1]
	locScale := regExScale.FindStringIndex(input)
	loc := regExOg.FindStringIndex(input)
	var opt string
	if !global.IntergratedTimeOut {
		opt = ""
	} else if locScale != nil {
		input = input[locScale[0]:locScale[1]]
		strScaleInd := regExNbre.FindStringIndex(input)
		strScaleVal := input[strScaleInd[0]:strScaleInd[1]]
		opt = input[:strScaleInd[0]]
		intVal, err := strconv.ParseInt(strScaleVal, 10, 64)
		if err != nil {
			log.Fatal(err)
		} else {
			timeOutVal := global.TimeOut * intVal
			opt = opt + fmt.Sprintf("%d", timeOutVal)
		}

	} else if loc != nil {
		opt = strings.Replace(input[loc[0]:loc[1]], "sec", fmt.Sprint(global.TimeOut), 1)
	} else {
		opt = ""
	}
	return opt
}

func InitBenchmark() (solverVersionMap map[string][][]string, benchmarkLogicMap map[string][][]string, timeOutOPtionMap map[string]string, supportedLogic map[string][]string) {

	regExComma := regexp.MustCompile(`,`)
	regExLine := regexp.MustCompile(`\n`)
	regExSquareBra := regexp.MustCompile(`\[.*?\]`)
	regExSpace := regexp.MustCompile(` `)
	regExLogic := regexp.MustCompile(`[A-Z_]+`)
	regExVersion := regexp.MustCompile(`[[:graph:]]+`)
	regExScale := regexp.MustCompile(`[[:graph:]]+[0-9]+( |)\*( |)sec`)
	regExOg := regexp.MustCompile(`[[:graph:]]+sec`)
	regExNbre := regexp.MustCompile(`[0-9]+`)

	rawData, err := os.ReadFile(global.ConfiFilePath)
	if err != nil {
		fmt.Printf("Configuration file at %s cannot be opened", global.ConfiFilePath)
		log.Fatal(err)
	}

	lineConfigFile := regExLine.Split(string(rawData), -1)

	timeOutOPtionMap = make(map[string]string)
	supportedLogic = make(map[string][]string)
	solverPathMap := make(map[string]string)
	versionMap := make(map[string][]string)
	var solverList []string
	var currLine string
	var currSolParam []string
	var logics []string
	var version []string
	var filteredVersion []string
	var currFilteredVersion string
	var filteredLogics []string
	var currFilteredLogic string
	var currSolName string
	var currSubLine []string
	var limEmptyLineConfigFile int = 5

	for i := 2; i < len(lineConfigFile); i++ {

		currLine = lineConfigFile[i]

		if currLine == "" {
			limEmptyLineConfigFile = limEmptyLineConfigFile - 1
		}

		if limEmptyLineConfigFile <= 0 {
			log.Print("Too many empty line in the config line, only the first block taken into account")
			break
		}

		currSubLine = regExSpace.Split(currLine, 2)
		currSolName = currSubLine[0]

		currSolParam = regExSquareBra.FindAllString(currSubLine[1], -1)

		// the []
		for j := 0; j < len(currSolParam); j++ {
			currSolParam[j] = currSolParam[j][1:(len(currSolParam[j]) - 1)]
		}

		solverList = append(solverList, currSolName)

		version = regExComma.Split(currSolParam[0], -1)
		for _, currVersion := range version {
			currFilteredVersion = regExVersion.FindString(currVersion)
			if len(currFilteredVersion) > 0 {
				filteredVersion = append(filteredVersion, currFilteredVersion)
			}
		}

		solverPathMap[currSolName] = currSolParam[1]

		logics = regExComma.Split(currSolParam[2], -1)

		for _, currLobgic := range logics {
			currFilteredLogic = regExLogic.FindString(currLobgic)
			if len(currFilteredLogic) > 0 {
				filteredLogics = append(filteredLogics, currFilteredLogic)
			}
		}

		//here problem if empty -> has null charceter nonethless insside find a way to clean input

		//words 2-3 for option

		timeOutOPtionMap[currSolName] = compileTimeoutOpt(regExScale, regExOg, regExNbre, currSolParam[3])

		supportedLogic[currSolName] = filteredLogics
		versionMap[currSolName] = filteredVersion
		filteredLogics = []string{}
		filteredVersion = []string{}
	}

	var benchmarkList []string
	for _, solver := range solverList {
		for _, logic := range supportedLogic[solver] {
			if !slices.Contains(benchmarkList, logic) {
				benchmarkList = append(benchmarkList, logic)
			}
		}
	}

	solverVersionMap = versionScan(solverPathMap, solverList, lineConfigFile[0], versionMap)
	benchmarkLogicMap = benchmarkScan(benchmarkList, lineConfigFile[1])

	return
}
