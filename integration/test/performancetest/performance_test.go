// go:build linux && integration
// +build linux,integration

package performancetest

import(
	"testing"
	"time"
	"log"
	"context"
	"fmt"
	"os"
	"github.com/aws/amazon-cloudwatch-agent/integration/test"
)

const (
	configPath = "resources/config.json"
	configOutputPath = "/opt/aws/amazon-cloudwatch-agent/bin/config.json"
	agentRuntimeMinutes = 5 //20 def
	

)

func TestPerformance(t *testing.T) {
	agentContext := context.TODO()
	instanceId := test.GetInstanceId()
	log.Printf("Instance ID used for performance metrics : %s\n", instanceId)

	test.CopyFile(configPath, configOutputPath)

	test.StartAgent(configOutputPath, true)
	fmt.Println("N_Logs",os.Getenv("PERFORMANCE_NUMBER_OF_LOGS"))
	agentRunDuration := agentRuntimeMinutes * time.Minute
	//let agent run before collecting performance metrics on it
	time.Sleep(agentRunDuration)
	log.Printf("Agent has been running for : %s\n", (agentRunDuration).String())
	test.StopAgent()

	//collect data
	data, err := GetPerformanceMetrics(instanceId, agentRuntimeMinutes, agentContext)
	if err != nil {
		log.Println("Error: ", err)
		t.Fatalf("Error: %v", err)
	}

	//------Placeholder to put data into database------//
	//useless code so data get used and compiler isn't mad
	if data == nil {
		t.Fatalf("No data")
	}
	//data base 
	dynamoDB := InitializeTransmitterAPI("CWAPerformanceMetrics") //add cwa version here
	if dynamoDB == nil{
		t.Fatalf("Error: generating dynamo table")
	}
	_, err = dynamoDB.SendItem(data)
	if err !=nil{
		t.Fatalf("Error: couldnt upload metric data to table")
	}
}
func TestUpdateCommit(t*testing.T){
	if(os.Getenv("IS_RELEASE") ==""){
		t.Skip("")
	}
	fmt.Println("Updating Release Commit")
	dynamoDB := InitializeTransmitterAPI("CWAPerformanceMetrics") //add cwa version here
	testHash := os.Getenv(SHA_ENV)
	if dynamoDB == nil{
		t.Fatalf("Error: generating dynamo table")
	return
	}
	//@TODO figure out how to get date
	dynamoDB.UpdateReleaseTag(2022,testHash)
}
