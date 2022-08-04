package windows_installation_test

import (
	"bytes"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

)
const (
	TIMEOUT  = 20 // 20*100= 2000 millisecond = 2second of trying
	MSI_NAME = "amazon-cloudwatch-agent.msi"
	MSIEXEC  = "msiexec.exe"
	PERIOD   = 100 // milliseconds. How often to check for shells
)

/*
TestShellCreation()
Desc: This test runs the msi the and checks if any powershell's parent is an msi.
This allows us to check if msi created a powershell child.
Fail: Msi created a powershell child
Success: Msi has not created any powershell children before timeout
*/
func TestShellCreation(t *testing.T) {
	oldPIDs := getCurrentPIDs() //take a screenshot of PID before installation
	run := exec.Command(MSIEXEC, "/i", MSI_NAME)
	if run == nil {
		t.Fatalf("ERROR: Couldn't run MSI")
	}
	err:=run.Start()
	if err !=nil{
		t.Fatalf("%s",err)
	}
	
	i:=0
	for !isMSIRunning() {
		if( i> TIMEOUT){
			break
		}
		i++;
	} //wait till process starts
	t.Log("Installation Started")
	for i = 0; i < TIMEOUT; i++ {
		newPIDs := getCurrentPIDs()
		if !reflect.DeepEqual(newPIDs, oldPIDs) { //check if there are any new PIDs
			diffPIDs := getDifference(oldPIDs, newPIDs)
			for _, pid := range diffPIDs {
				t.Logf("PID in diffPIds %s",pid)
				parent := getParent(pid)
				if parent == MSIEXEC {
					t.Fatalf("ERROR: MSI summoned powershell window")
				}
			}
			oldPIDs = newPIDs
		}
		time.Sleep(PERIOD * time.Millisecond)
		if isMSIRunning() {
			t.Log("Installation Completed",i)
			break
		}
	}
}

/*
getParent()
Desc: This function gets the parent of an process' pid
Param:
	pid:string; process id of the process in string form
Return: parent's process id in string form
*/
func getParent(pid string) string {
	parentPidSearchConfig := map[string]string{"ProcessId": pid}
	parentPid := QueryProcessAttribute(parentPidSearchConfig, "ParentProcessId")
	fmt.Println(pid, parentPid)
	parentProcessSearchConfig := map[string]string{"ProcessId": parentPid}
	parent := QueryProcessAttribute(parentProcessSearchConfig, "Caption")
	return parent
}

/*
getCurrentPIDs()
Desc: This function returns the pid's of running powershells
Param: None
Return: List of PIDs in string form
*/
func getCurrentPIDs() []string {
	preInstallerShells := []string{}
	processes := getCurrentProcesses()
	for _, val := range processes {
		if strings.Contains(val, "PID") {
			pid := strings.Split(val, "          ")[1]
			// fmt.Println(pid)
			if !contains(preInstallerShells, pid) {
				preInstallerShells = append(preInstallerShells, pid)
			}
		}
	}
	return preInstallerShells
}

/*
getCurrentProcesses()
Desc: This returns list of powershells running
Param: None
Return: List of powershells
*/
func getCurrentProcesses() []string {
	cmd := exec.Command("tasklist", "/FI", "ImageName eq powershell.exe", "/FI", "Status eq Running", "/FO", "LIST")
	rawTags, _ := cmd.Output()
	cmds := strings.Split(string(rawTags), "\r\n")
	return cmds

}

/*
isMSIRunning()
Desc: This function checks if an msiexec.exe is running
Param: None
Return: If MSIexec is running or not
*/
func isMSIRunning() bool {
	errorString := "INFO: No tasks are running which match the specified criteria."
	cmd := exec.Command("tasklist", "/FI", "ImageName eq msiexec.exe", "/FI", "Status eq Running", "/FO", "LIST")
	rawTags, _ := cmd.Output()
	cmds := strings.Split(string(rawTags), "\r\n")
	// fmt.Println(cmds,len(cmds))
	return !contains(cmds, errorString)
}

/*
getPID()
Desc: This function gets PID of a process from its name
Param:
	processName:string; the name of the process like powershell.exe
Return:
*/
func getPID(processName string) string {
	cmd := exec.Command("tasklist", "/FI", "ImageName eq "+processName+".exe", "/FI", "Status eq Running", "/FO", "LIST")
	rawTags, _ := cmd.Output()
	cmds := strings.Split(string(rawTags), "\r\n")

	for _, val := range cmds {
		if strings.Contains(val, "PID") {
			return strings.Split(val, "          ")[1]
		}
	}
	return ""
}

/*
QueryProcessAttribute()
Desc: This function searches process for a specific attribute and returns a specific attribute
Param:
	searchAttribute: This map contains what search attribute like {"ProcessId":"1000"}
	returnAttribute: The attribute of process you want returned. "ParentProcessId"
Return: A string type attribute, depending on  returnAttribute
*/
func QueryProcessAttribute(searchAttributes map[string]string, returnAttribute string) string {
	expressionList := []string{}
	for attributeKey, attributeValue := range searchAttributes {
		expressionList = append(expressionList, fmt.Sprintf("%s=%s", attributeKey, attributeValue))
	}
	searchExpression := strings.Join(expressionList, " and")
	cmd := exec.Command("wmic", "process", "where", searchExpression, "get", returnAttribute)
	var outb bytes.Buffer
	cmd.Stdout = &outb
	err := cmd.Run()

	cleanOutput := wmicParser(outb.String())
	fmt.Println(len(cleanOutput), cleanOutput, err)
	if len(cleanOutput) == 0 {
		return ""
	}
	return cleanOutput[1]

}

/*
wmicParser()
Desc: This function parses the outout of wmic commands.
This needed because wmic commands have compilcated string outputs.
Param:
	out:string, the output of wmic command
Return: a list string where 0 index is the attribute,
and the remaining is the outputs of the wmic command
*/
func wmicParser(out string) []string {
	const CR = '\r'
	const NL = '\n'
	const SP = ' '
	var output []string
	crCount := 0
	word := ""
	for _, char := range out {
		if char == CR {
			crCount++
			continue
		}
		if crCount == 2 {
			crCount = 0

			output = append(output, word)
			word = ""
		}
		if char != NL && char != SP {
			word += string(char)
		}
	}
	return output
}

/*
contains()
Desc: This is element checks if a string is part of a slice
Param:
	slice: string slice
	item: string
Return: bool depending on if the item is in the slice
*/
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if item == v {
			return true
		}
	}
	return false
}

/*
getDifference()
Desc: This function returns the different elements between a old and a new slice.
Param:
	oldSlice: the previous version of a slice
	newSlice: the update version of a slice
Return: A slice of different elements between  two slices
*/
func getDifference(oldSlice []string, newSlice []string) []string {
	differenceSet := make(map[string]bool) //set
	diffPIDs := []string{}
	for _, val := range oldSlice {
		if !differenceSet[val] {
			differenceSet[val] = true
		}
	}
	for _, val := range newSlice {
		if !differenceSet[val] {
			diffPIDs = append(diffPIDs, val)
		}
	}
	return diffPIDs

}
