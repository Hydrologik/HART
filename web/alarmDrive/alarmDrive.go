package main

import (
	"HART/web/clientTag"
	"HART/web/mongoDrive"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

const longForm = "Jan 2, 2006 at 3:04pm (MST)"

type IgnAlarmFunc func(mongoDrive.Alert, map[string]interface{}, string) (bool, error)

func noData(a mongoDrive.Alert, val map[string]interface{}, flag string) (bool, error) {
	_, ok := val["Value"].(float64)
	return ok, nil
	//May add functionality to
}

func threshold(a mongoDrive.Alert, val map[string]interface{}, flag string) (bool, error) {
	v, ok := val["Value"].(float64)
	if !ok {
		//Alarming on a nil doesnt make sense and should be setting off noData alarm
		return true, nil
	}

	var t float64

	switch flag {
	case "h":
		t, ok = a.Config["High"].(float64)
		if !ok {
			return false, nil
		}
		return v < t, nil
	case "l":
		t, ok := a.Config["Low"].(float64)
		if !ok {
			return false, nil
		}
		return v < t, nil
	default:
		//How the fuck to we get here!?!
		return true, nil
	}
}

func staleData(a mongoDrive.Alert, val map[string]interface{}, flag string) (bool, error) {
	v, ok := val["Value"].(float64)
	var c int
	//No value doesnt make sense to check
	if !ok {
		return true, nil
	}
	ov, ok := a.Config["val"].(float64)
	if !ok {
		a.Config["val"] = v
	}

	c, ok = a.Config["count"].(int)
	if !ok {
		a.Config["count"] = 0
		c = 0
	}

	if ov == v {
		c++
		if c > a.Config["CountThresh"].(int) {
			a.Config["count"] = c
			return false, nil
		}
		a.Config["count"] = c
		return true, nil
	} else {
		a.Config["val"] = v
		a.Config["count"] = 0
		return true, nil
	}
}

func df(a mongoDrive.Alert, val map[string]interface{}, flag string) (bool, error) { return false, nil }

func alertEmail(a mongoDrive.Alert) error { return nil }

// Runs alarm list update status and send alert where needed
func RunIgnAlerts(ignData map[string]interface{}) error {
	alarms, err := mongoDrive.GetIgnAlarms(bson.D{})
	if err != nil {
		return err
	}
	var afunc IgnAlarmFunc
	f := ""
	for _, alarm := range alarms {

		switch alarm.Type {
		case "BadNoData":
			afunc = noData
		case "HighVal":
			f = "h"
			afunc = threshold
		case "LowVal":
			f = "l"
			afunc = threshold
		case "StaleVal":
			afunc = threshold
		default:
			afunc = df
		}

		tagVal, ok := ignData[alarm.Client].(map[string]interface{})[alarm.Site].(map[string]interface{})[alarm.Tag].(map[string]interface{})
		if !ok {
			fmt.Printf("Alarm returned nil from ign data:\n%s", alarm)
			continue
		}
		valid, err := afunc(alarm, tagVal, f)
		if err != nil {
			return err
		}

		switch alarm.State {
		case "Good":
			//Previous good and no alert
			if valid {
				alarm.ObsvCount++
				//At creation time date string = ""
				if alarm.EntryDate == "" {
					alarm.EntryDate = time.Now().Format(longForm)
				}
			} else {
				//Previous good but now error
				alarm.ObsvCount = 1
				alarm.State = "Warn"
				alarm.EntryDate = time.Now().Format(longForm)
			}
		case "Warn":
			//Previous in Warn no alert
			if valid {
				alarm.ObsvCount = 1
				alarm.State = "Good"
				alarm.EntryDate = time.Now().Format(longForm)
			} else {
				alarm.ObsvCount++
				//Past alarm threshold
				if alarm.ObsvCount > alarm.Threshold {
					alarm.State = "Alert"
					alarm.EntryDate = time.Now().Format(longForm)
					alertEmail(alarm)
				}
			}
		case "Alert":
			//Previous in Alert but corrected
			if valid {
				alarm.ObsvCount = 1
				alarm.State = "Good"
				alarm.EntryDate = time.Now().Format(longForm)
			} else {
				//Was in alert and still is
				//Send alert every week after entry date
				//May or may not implement
				//alertEmail(alarm)
				alarm.ObsvCount++
			}
		}

		err = mongoDrive.EditIgnAlarm(alarm)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	data, _ := clientTag.IgnCall()
	RunIgnAlerts(data)
}
