package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"HART/web/mongoDrive"
)

type Card struct {
	Name       string
	ChildCount int
	Alert      int
	Warn       int
	Good       int
}

func GetKeyList(m map[string]interface{}) []string {
	var keyList []string
	for key, chld := range m {
		if len(chld.(map[string]interface{})) > 0 {
			keyList = append(keyList, key)
		}

	}
	return keyList
}

/*
Fancy print the client list
Function more as a utliilty than requirement
*/
func fprint(result map[string]interface{}) {
	for key, val := range result {
		if len(result[key].(map[string]interface{})) > 0 {
			fmt.Printf("%s\n", key)
			for k, v := range val.(map[string]interface{}) {
				fmt.Printf("\t%s\n", k)
				for i, j := range v.(map[string]interface{}) {
					fmt.Printf("\t\t%s\n", i)
					jm := j.(map[string]interface{})
					fmt.Printf("\t\t\tQuality: %s, Value: %s, timestamp: %s\n", jm["Quality"].(string), jm["Value"], jm["Timestamp"].(string))
					for m, d := range j.(map[string]interface{}) {
						fmt.Printf("\t\t\t%s : %s\n", m, d)
					}
				}
			}
			fmt.Println()
		}
	}
}

func IgnCall() (map[string]interface{}, error) {
	url := "https://hydrologik.cloud:443/system/ws/rest/ignitionAgent"
	cl := &http.Client{}
	var out map[string]interface{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("token", "SunWod4117!")
	res, err := cl.Do(req)
	if err != nil {
		return map[string]interface{}{}, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return map[string]interface{}{}, err
	}

	err = json.Unmarshal(body, &out)
	if err != nil {
		return map[string]interface{}{}, err
	}

	result := out["content"].(map[string]interface{})
	return result, nil
}

func GetCardList(level string, m map[string]interface{}, c string, s string) ([]Card, error) {
	var crdLst []Card
	var slct map[string]interface{}

	//In the switch we set the level of query are we getting
	//Client cards, Site Cards or Tag Cards
	var mongStr string
	quryStr := []string{c, s, ""}
	var i int

	//Dig Downs from client to site to tag will be very similar
	//We use a switch to reduce code duplication
	switch level {
	case "client":
		slct = m
		mongStr = "client"
		i = 0
	case "site":
		slct = m[c].(map[string]interface{})
		mongStr = "site"
		i = 1
	case "tag":
		slct = m[c].(map[string]interface{})[s].(map[string]interface{})
		mongStr = "tag"
		i = 2
	}

	for key, child := range slct {
		if len(child.(map[string]interface{})) > 0 {
			quryStr[i] = key
			met, err := mongoDrive.GetAlerMetrics(mongStr, quryStr[0], quryStr[1], quryStr[2])
			if err != nil {
				return nil, err
			}
			newCd := Card{
				Name:       key,
				ChildCount: len(child.(map[string]interface{})),
				Alert:      met.Alert,
				Warn:       met.Warn,
				Good:       met.Good,
			}
			crdLst = append(crdLst, newCd)
		}
	}

	return crdLst, nil
}

func main() {
	data, err := IgnCall()
	if err != nil {
		log.Fatal(err.Error())
	}

	cardLst, err := GetCardList("client", data, "", "")
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println(cardLst)
}
