//package main
package mongoDrive

import (
	"context"
	"fmt"
	"log"
	"os"
	

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Alert struct {
	Client    string
	Site      string
	Tag       string
	AlertType string
	Config   map[string]interface{}
	State     string
	EntryDate string
	ObsvCount int
	Threshold int
	Emails    []string
}

type AlertMetric struct {
	Alert int
	Warn  int
	Good  int
}

func getClient() *mongo.Client {
	//env file for sensative data and basic Aut
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
		panic(err)
	}
	uri := os.Getenv("MONSTR")

	//Open Client
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	return client

}

//TODO:
//func AddAlert(a Alert) error{}

//func GetAlerts() ([]Alert, error){}

//func EditAlert(id string) error{}


//Function takes in level of inqury and corresponding string specification
//Creates new mongoClient and calls Alerts db to count responses and return metrics
func GetAlerMetrics(lvl string, c string, s string, t string) (AlertMetric, error) {
	met := AlertMetric{
		Alert: 0,
		Warn: 0,
		Good: 0,	
	}
	var filter bson.D
	switch lvl {
	case "client":
		filter = bson.D{{"Client", c}}
	case "site":
		filter = bson.D{{"Client", c}, {"Site", s}}
	case "tag":
		filter = bson.D{{"Client", c}, {"Site", s}, {"Tag", t}}
	default:
		filter = bson.D{{}}
	}
	client := getClient()
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	coll := client.Database("Alerts").Collection("Reports")
	var res []Alert
	cur, err := coll.Find(context.TODO(), filter)
	if err != nil{
		return AlertMetric{}, err
	}

	if err = cur.All(context.TODO(), &res); err != nil{
		return AlertMetric{}, err
	}

	for _, a := range(res){
		switch a.State{
		case "Alert":
			met.Alert ++
		case "Warn":
			met.Warn ++
		case "Good":
			met.Good ++
		}
	}

	return met, nil
}

/*
func main() {
	metrics, err := GetAlerMetrics("client", "Conejos", "", "")
	if err != nil{
		log.Fatal(err.Error())
	}

	fmt.Println(metrics)
}
*/