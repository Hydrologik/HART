// package main
package mongoDrive

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
)

type Alert struct {
	Client    string
	Site      string
	Tag       string
	Type      string
	Config    map[string]interface{}
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

func AddIgnAlarm(a Alert) error {
	client := getClient()
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	coll := client.Database("Alerts").Collection("Ignition")
	_, err := coll.InsertOne(context.TODO(), a)
	if err != nil {
		return err
	}
	return nil
}

func GetIgnAlarms(filter bson.D) ([]Alert, error) {
	client := getClient()
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	coll := client.Database("Alerts").Collection("Ignition")
	var res []Alert
	cur, err := coll.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	if err = cur.All(context.TODO(), &res); err != nil {
		return nil, err
	}

	//Sort alpabetically by client name
	slices.SortStableFunc(res, func(a, b Alert) int {
		return cmp.Compare(a.Client, b.Client)
	})

	return res, nil

}

// Edit alarm takes in alarm edits struct and replaces the current alarm in db
func EditIgnAlarm(a Alert) error {
	client := getClient()
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	coll := client.Database("Alerts").Collection("Ignition")
	filter := bson.D{{Key: "client", Value: a.Client}, {Key: "site", Value: a.Site}, {Key: "tag", Value: a.Tag}, {Key: "type", Value: a.Type}}
	_, err := coll.ReplaceOne(context.TODO(), filter, a)
	if err != nil {
		return err
	}
	return nil
}

// Delete ign alarm takes in a single alarm filter and deletes from db
func DeleteIgnAlarm(f bson.D) error {
	client := getClient()
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	coll := client.Database("Alerts").Collection("Ignition")
	dc, err := coll.DeleteOne(context.TODO(), f)
	if err != nil {
		return err
	}
	if dc.DeletedCount != 1 {
		return errors.New("delete failed to find alarm to delete")
	}
	return nil
}

// Function takes in level of inqury and corresponding string specification
// Creates new mongoClient and calls Alerts db to count responses and return metrics
func GetIgnMetrics(lvl string, c string, s string, t string) (AlertMetric, error) {
	met := AlertMetric{
		Alert: 0,
		Warn:  0,
		Good:  0,
	}
	var filter bson.D
	switch lvl {
	case "client":
		filter = bson.D{{Key: "client", Value: c}}
	case "site":
		filter = bson.D{{Key: "client", Value: c}, {Key: "site", Value: s}}
	case "tag":
		filter = bson.D{{Key: "client", Value: c}, {Key: "site", Value: s}, {Key: "tag", Value: t}}
	default:
		filter = bson.D{{}}
	}

	res, err := GetIgnAlarms(filter)
	if err != nil {
		return AlertMetric{}, err
	}

	for _, a := range res {
		switch a.State {
		case "Alert":
			met.Alert++
		case "Warn":
			met.Warn++
		case "Good":
			met.Good++
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
