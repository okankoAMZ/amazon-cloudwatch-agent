package transmitter

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)
const(
	TIME_CONSTANT = 300.0 // this const is in seconds , 5 mins
)
type TransmitterAPI struct{
	DynamoDbClient dynamodbiface.DynamoDBAPI
	DataBaseName      string // this is the name of the table when test is run
}
// this is the packet that will be sent converted to DynamoItem
type Metric struct{
	Average float64
	StandardDev float64
	Period	int 	//in seconds
	Data []float64
}

type collectorData []struct{ // this is the struct data collector passes in
	Id string	`json:"Id"`
	Label string	`json:Label`	
	Messages string	`json:Messages`
	StatusCode string	`json:StatusCode`
	Timestamps []string	`json:Timestamps`
	Values []float64	`json:Values`
}


/*
InitializeTransmitterAPI
Desc: Initializes the transmitter class
Side effects: Can create a dynamodb table
*/
func InitializeTransmitterAPI(DataBaseName string) * TransmitterAPI{
	transmitter := new(TransmitterAPI)
	//setup aws session
	session := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	transmitter.DynamoDbClient =  dynamodb.New(session)
	transmitter.DataBaseName = DataBaseName //fmt.Sprintf("%d",int(time.Now().UnixNano()))
	// check if the dynamo table exist if not create it
	tableExist, err:= transmitter.TableExist()
	if err !=nil{
		return nil
	}
	if !tableExist{
		fmt.Println("Table doesn't exist")
		err := transmitter.CreateTable()
		if err != nil{
			fmt.Println("Couldn't create table")
			return nil
		}
	}
	fmt.Println("API ready")
	return transmitter

}
/*
CreateTable()
Desc: Will create a DynamoDB Table with given param. and config
Params:
Side Effects: Creates a dynamoDB table
*/
func (transmitter * TransmitterAPI) CreateTable() error{
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Hash"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Hash"),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
		TableName: aws.String(transmitter.DataBaseName),
	} // this is the config for the new table
	
	_, err := transmitter.DynamoDbClient.CreateTable(input)
	if err != nil {
		fmt.Printf("Got error calling CreateTable: %s", err)
		return err
	}
	
	fmt.Println("Created the table", transmitter.DataBaseName)
	return nil
}
/*
AddItem() 
Desc: Takes in a packet and
will convert to dynamodb format  and upload to dynamodb table.
Param: 
	packet * map[string]interface{}:  is a map with data collection data
Side effects: 
	Adds an item to dynamodb table
*/
func (transmitter * TransmitterAPI) AddItem(packet map[string]interface{})(string,error){
	// fmt.Printf("Packet: %+v \n",packet)
	// metrics, _ := dynamodbattribute.MarshalMap(packet.Metrics)
	// fmt.Printf("Metric: %+v \n OG:%+v \n",metrics, packet.Metrics)
	item, err := dynamodbattribute.MarshalMap(packet)
	if err != nil {
		panic(err)
	}
	DBitem := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(transmitter.DataBaseName),
	}
	// fmt.Println(DBitem)
	// return err
	_, err = transmitter.DynamoDbClient.PutItem(DBitem)
	if err != nil {
		fmt.Printf("Couldn't add item to table. Here's why: %v\n", err)
	}
	return fmt.Sprintf("%v",item),err
}
/*
TableExist()
Desc: Checks if the the table exist and returns the value
//https://github.com/awsdocs/aws-doc-sdk-examples/blob/05a89da8c2f2e40781429a7c34cf2f2b9ae35f89/gov2/dynamodb/actions/table_basics.go
*/
func (transmitter * TransmitterAPI) TableExist() (bool,error){
	l,err := transmitter.ListTables()
	for i:=0; i< len(l); i++{
		if transmitter.DataBaseName == l[i]{
			return true,nil
		}
	}
	return false,err

}
/*
RemoveTable()
Desc: Removes the table that was craeted with initialization. 
Side effects: Removes a Dynamodb table
*/
func (transmitter * TransmitterAPI) RemoveTable() error{
	input := dynamodb.DeleteTableInput{TableName: aws.String(transmitter.DataBaseName)}

	_,err:=transmitter.DynamoDbClient.DeleteTable(&input)
	if err !=nil{
		fmt.Println(err)
	}
	return err
}
// ListTables lists the DynamoDB table names for the current account.
func (transmitter * TransmitterAPI) ListTables() ([]string, error) {
	var tableNames []string
	input := &dynamodb.ListTablesInput{}
	tables, err := transmitter.DynamoDbClient.ListTables(
		input)
	if err != nil {
		fmt.Printf("Couldn't list tables. Here's why: %v\n", err)
	} else {
		for  i:=0; i< len(tables.TableNames); i++{
			tableNames =  append(tableNames,*tables.TableNames[i])
		}
		
	}
	return tableNames, err
}

func (transmitter * TransmitterAPI) SendItem(data []byte) (string,error) {
	// return nil
	packet, err := transmitter.Parser(data)
	if err != nil{
		return "",err
	}
	sentItem,err := transmitter.AddItem(packet) 
	return sentItem,err
}

func (transmitter * TransmitterAPI) Parser(data []byte) (map[string]interface{},error){
	dataHolder := collectorData{}
	err := json.Unmarshal(data,&dataHolder)
	if err !=nil{
		return nil,err
	}
	packet := make(map[string]interface{})
	//temp solution
	packet["Hash"] = fmt.Sprintf("%d",time.Now().UnixNano())
	/// will remove
	for _,rawMetricData:= range dataHolder{
		numDataPoints := float64(len(rawMetricData.Timestamps))
		sum :=0.0
		for _,val := range rawMetricData.Values {
			sum += val
		}
		avg :=  sum /numDataPoints
		// calculate diff between mean and values 
		diffSum := 0.0
		for _,val := range rawMetricData.Values{
			diffSum = diffSum +  (avg - val)
		}
		
		metric := Metric{
			Average: avg,
			StandardDev: math.Sqrt(math.Pow(diffSum,2)/float64(numDataPoints)),
			Period: int(TIME_CONSTANT/(numDataPoints)),
			Data:  rawMetricData.Values}
		// fmt.Printf("%+v\n",metric)
		// packet.Metrics[rawMetricData.Label] = metric
		packet[rawMetricData.Label] = metric

		
	}
	return packet,nil
}

func testFiller(testname string) * TransmitterAPI{
	transmitter := new(TransmitterAPI)
	//setup aws session
	session := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	transmitter.DynamoDbClient =  dynamodb.New(session)
	transmitter.DataBaseName = testname
	return transmitter
}