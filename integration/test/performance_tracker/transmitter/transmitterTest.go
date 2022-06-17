package transmitter
import (
	"testing"
	"time"
	"fmt"
	"os"
	"os/exec"
)
/*
TestTableGeneration
Desc: This test checks if a dynamodb table can be created succesfully
Check: After creating the table it checks by checking if it is listed in all tables.
Fail: It can fail if table cannot be create or removed
*/
func TestTableGeneration(t * testing.T){
	testname  := "TestTableGeneration"
	transmitter := testFiller(testname)
	transmitter.DataBaseName = testname

	err := transmitter.CreateTable()
	if err != nil{
		t.Errorf("Couldn't create table")
	}

	time.Sleep(7 * time.Second) // gives time for table generation to compelete
	l,_ := transmitter.ListTables()
	t.Log(l)
	//add cleanup
	err = transmitter.RemoveTable()
	if err !=nil{
		t.Errorf("Couldnt remove the table")
	}
	time.Sleep(2 * time.Second)

}
/*
TestTransmitterInit
Desc: This test checks if a transmitter class can be initialized succesfully
Check: After initialization it checks if the object is nil or not
Fail: If the object is nil it fails
*/
func TestTransmitterInit(t * testing.T){
	transmitter := InitializeTransmitterAPI("TestTransmitterInit")
	if transmitter == nil{
		t.Errorf("Couldnt generate transmitter")
	}else if transmitter.DynamoDbClient == nil{
		t.Errorf("Couldnt generate transmitter")
	}
	time.Sleep(10 * time.Second)
	transmitter.RemoveTable()
}
/*
TestParser
Desc: This test checks if a transmitter parser can succesfully parse and create a struct.
Check: Checks if map was created.
Fail: If the object is nil it fails
*/
func TestParser(t * testing.T){
	filedata ,_ := os.ReadFile("../data_collector/data.json")
	// t.Errorf(string(filedata))
	transmitter :=testFiller("TestParser")
	parsedText,err:= transmitter.Parser(filedata)
	if err !=nil{
		t.Errorf("Couldnt parse")
	}
	fmt.Printf("%+v\n",parsedText)
}
