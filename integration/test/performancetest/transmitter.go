package performancetest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const METRIC_PERIOD = 300.0 // this const is in seconds , 5 mins

type TransmitterAPI struct {
	dynamoDbClient *dynamodb.Client
	DataBaseName   string // this is the name of the table when test is run
}

// this is the packet that will be sent converted to DynamoItem
type Metric struct {
	Average float64
	P99     float64 //99% percent process
	Max     float64
	Min     float64
	Period  int //in seconds
	Data    []float64
}

type collectorData []struct { // this is the struct data collector passes in
	Id         string    `json:"Id"`
	Label      string    `json:Label`
	Messages   string    `json:Messages`
	StatusCode string    `json:StatusCode`
	Timestamps []string  `json:Timestamps`
	Values     []float64 `json:Values`
}

/*
InitializeTransmitterAPI
Desc: Initializes the transmitter class
Side effects: Creates a dynamodb table if it doesn't already exist
*/
func InitializeTransmitterAPI(DataBaseName string) *TransmitterAPI {
	//setup aws session
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		fmt.Printf("Error: Loading in config %s\n", err)
	}
	transmitter := TransmitterAPI{
		dynamoDbClient: dynamodb.NewFromConfig(cfg),
		DataBaseName:   DataBaseName,
	}
	// check if the dynamo table exist if not create it
	tableExist, err := transmitter.TableExist()
	if err != nil {
		return nil
	}
	if !tableExist {
		fmt.Println("Table doesn't exist")
		err := transmitter.CreateTable()
		if err != nil {
			return nil
		}
	}
	fmt.Println("API ready")
	return &transmitter

}

/*
CreateTable()
Desc: Will create a DynamoDB Table with given param. and config
*/
 //add secondary index space vs time  
func (transmitter *TransmitterAPI) CreateTable() error {
	_, err := transmitter.dynamoDbClient.CreateTable(
		context.TODO(), &dynamodb.CreateTableInput{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("Hash"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("Hash"),
					KeyType:       types.KeyTypeHash,
				},
			},
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(10),
				WriteCapacityUnits: aws.Int64(10),
			},
			TableName: aws.String(transmitter.DataBaseName),
		}) // this is the config for the new table)
	if err != nil {
		fmt.Printf("Error calling CreateTable: %s", err)
		return err
	}
	time.Sleep(10*time.Second)
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
func (transmitter *TransmitterAPI) AddItem(packet map[string]interface{}) (string, error) {
	item, err := attributevalue.MarshalMap(packet)
	if err != nil {
		panic(err)
	}
	_, err = transmitter.dynamoDbClient.PutItem(context.TODO(),
		&dynamodb.PutItemInput{
			Item:      item,
			TableName: aws.String(transmitter.DataBaseName),
		})
	if err != nil {
		fmt.Printf("Error adding item to table.  %v\n", err)
	}
	return fmt.Sprintf("%v", item), err
}

/*
TableExist()
Desc: Checks if the the table exist and returns the value
//https://github.com/awsdocs/aws-doc-sdk-examples/blob/05a89da8c2f2e40781429a7c34cf2f2b9ae35f89/gov2/dynamodb/actions/table_basics.go
*/
func (transmitter *TransmitterAPI) TableExist() (bool, error) {
	// l,err := transmitter.ListTables()
	// for i:=0; i< len(l); i++{
	// 	if transmitter.DataBaseName == l[i]{
	// 		return true,nil
	// 	}
	// }
	// return false,err
	exists := true
	_, err := transmitter.dynamoDbClient.DescribeTable(
		context.TODO(), &dynamodb.DescribeTableInput{TableName: aws.String(transmitter.DataBaseName)},
	)
	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			fmt.Printf("Table %v does not exist.\n", transmitter.DataBaseName)
			err = nil
		} else {
			fmt.Printf("Couldn't determine existence of table %v. Here's why: %v\n", transmitter.DataBaseName, err)
		}
		exists = false
	}
	return exists, err
}

/*
RemoveTable()
Desc: Removes the table that was craeted with initialization.
*/
func (transmitter *TransmitterAPI) RemoveTable() error {
	_, err := transmitter.dynamoDbClient.DeleteTable(context.TODO(),
		&dynamodb.DeleteTableInput{TableName: aws.String(transmitter.DataBaseName)})
	if err != nil {
		fmt.Println(err)
	}
	return err
}

/*
SendItem()
Desc: Parses the input data and adds it to the dynamo table
Param: data []byte is the data collected by data collector
*/
func (transmitter *TransmitterAPI) SendItem(data []byte) (string, error) {
	// return nil
	packet, err := transmitter.Parser(data)
	if err != nil {
		return "", err
	}
	sentItem, err := transmitter.AddItem(packet)
	return sentItem, err
}

func (transmitter *TransmitterAPI) Parser(data []byte) (map[string]interface{}, error) {
	dataHolder := collectorData{}
	err := json.Unmarshal(data, &dataHolder)
	if err != nil {
		return nil, err
	}
	packet := make(map[string]interface{})
	//@TODO: add git integration temp solution
	packet["Hash"] =  os.Getenv("SHA") //fmt.Sprintf("%d", time.Now().UnixNano())
	packet["CommitDate"] = os.Getenv("SHA_DATE")//fmt.Sprintf("%d", time.Now().UnixNano())
	/// will remove
	for _, rawMetricData := range dataHolder {
		numDataPoints := float64(len(rawMetricData.Timestamps))
		// @TODO:ADD GetMetricStatistics after merging with data collection code
		sum :=0.0
		for _,val := range rawMetricData.Values {
			sum += val
		}
		avg :=  sum /numDataPoints
		//----------------
		metric := Metric{
			Average: avg,
			Max:     100.0,
			Min:     0.0,
			P99:     0.0,
			Period:  int(METRIC_PERIOD / (numDataPoints)),
			Data:    rawMetricData.Values}
		packet[rawMetricData.Label] = metric
	}
	return packet, nil
}
