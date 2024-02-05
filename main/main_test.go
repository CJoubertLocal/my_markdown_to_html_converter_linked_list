package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
)

var pathToTestTextFiles = "../test_files/"
var pathToTestSolutionFiles = "../solution_files/"

type testCase struct {
	filePathAndName string
	expectedOutput  string
}

func TestRun(t *testing.T) {
	var testCases = []testCase{}

	files, err := os.ReadDir(pathToTestTextFiles)
	if err != nil {
		log.Fatal("unable to find test files:", err)
	}

	for _, file := range files {
		solution, err := os.ReadFile(pathToTestSolutionFiles + "solution_" + file.Name())
		// The HTML will work regardless of new line characters between tags.
		solutionString := strings.ReplaceAll(string(solution), "\r\n", "")
		if err != nil {
			log.Fatal("unable to find solution file for file name:", file.Name())
		}
		newTestCase := testCase{
			filePathAndName: pathToTestTextFiles + file.Name(),
			expectedOutput:  "<div class=\"content\">" + solutionString + "</div>", // NOTE: This is required for the output to be correct, but was not added until later on.
		}
		testCases = append(testCases, newTestCase)
	}

	for i, tst := range testCases {
		res := convertMarkdownFileToBlogHTML(tst.filePathAndName)
		res = strings.ReplaceAll(res, "\r\n", "")
		if res != tst.expectedOutput {
			t.Error(
				fmt.Sprintf("test %d, file %s: \nexpected: \n'%s' \nbut got \n'%s'",
					i, tst.filePathAndName, tst.expectedOutput, res,
				),
			)
		}
	}
}
