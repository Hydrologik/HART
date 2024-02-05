package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Tag struct {
	TagPath   string
	Timestamp string
	Quality   string
	Value     interface{}
}

type Station struct {
	Tags map[string]Tag
}

type Client struct {
	Stations map[string]Station
}


/*
Fancy print the client list
Function more as a utliilty than requirement
*/
func Fprint(s map[string]Client) {
	for cli, sta := range s {
		fmt.Printf("%s\n", cli)
		for stn, tag := range sta.Stations {
			fmt.Printf("\t%s\n", stn)
			for tgn, tgv := range tag.Tags {
				fmt.Printf("\t\t%s: Timestamp: %s, Quality: %s, Value: %s\n", tgn, tgv.Timestamp, tgv.Quality, tgv.Value)
			}
		}
		fmt.Println()
	}
}

func GetTags() {
	url := "https://hydrologik.cloud:443/system/ws/rest/ignitionAgent"
	cl := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("token", "SunWod4117!")
	res, _ := cl.Do(req)

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(string(body))

	var out map[string]interface{}

	err = json.Unmarshal(body, &out)
	if err != nil {
		fmt.Println(err)
	}

	result := out["content"].(map[string]interface{})

	for key, val := range result {
		if len(result[key].(map[string]interface{})) > 0{
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

	//Fprint(out.Content)

}

func main() {
	GetTags()
}
