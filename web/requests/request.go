package requests

import (
	"clientTag"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"
)



/*
Returns a sorted list of Clients, the Stations they hold, and the metric tags we keep history off
Sorting is alphabetical
Final structure is
List[Clients[Stations[Tags{Path, TS, Q}]]]
*/
func parseTagRaw(tagRaw map[string]interface{}) []clientTag.Client {

	var clientList []clientTag.Client

	for client := range tagRaw {
		tags := tagRaw[client].(map[string]interface{})
		adClient, adTag := false, false
		newClient := clientTag.Client{ClientName: client}
		//If we have valid tags
		if len(tags) != 0 {
			adClient = true
			//Loop throuh the tags
			for tag := range tags {
				metrics := tags[tag].(map[string]interface{})
				newStation := clientTag.Station{StationName: tag}
				//If we have metrics for the Tag
				if len(metrics) != 0 {
					adTag = true
					//Loop over the metrics
					for metric := range metrics {
						timeStamp, _ := time.Parse("Mon Jan _2 15:04:05 MST 2006", metrics[metric].(map[string]interface{})["Timestamp"].(string))
						newTag := clientTag.Tag{
							TagPath:   metric,
							Timestamp: timeStamp,
							Quality:   metrics[metric].(map[string]interface{})["Quality"].(string),
						}
						newStation.Tags = append(newStation.Tags, newTag)
					}
					//Sort tag paths
					sort.SliceStable(newStation.Tags,
						func(i, j int) bool { return newStation.Tags[i].TagPath < newStation.Tags[j].TagPath })
					newClient.Stations = append(newClient.Stations, newStation)
				}
			}
			if adClient && adTag {
				//Sort Station names
				sort.SliceStable(newClient.Stations,
					func(i, j int) bool { return newClient.Stations[i].StationName < newClient.Stations[j].StationName })
				clientList = append(clientList, newClient)
			}

		}
	}
	//Sort by client name
	sort.SliceStable(clientList,
		func(i, j int) bool { return clientList[i].ClientName < clientList[j].ClientName })
	return clientList
}

func GetTagRaw() ([]clientTag.Client, error) {
	apiEndpoint := "https://hydrologik.cloud:443/system/ws/rest/ignitionAgent"
	var tagRaw map[string]interface{}
	client := &http.Client{}
	req, _ := http.NewRequest("GET", apiEndpoint, nil)

	//TODO: Fill this in with .env variables
	req.Header.Set("token", "SunWood4117!")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close() //Close body after function execution
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, &json.SyntaxError{}
	}

	err = json.Unmarshal(body, &tagRaw)
	if err != nil {
		return nil, err
	}

	return parseTagRaw(tagRaw["content"].(map[string]interface{})), nil
}
