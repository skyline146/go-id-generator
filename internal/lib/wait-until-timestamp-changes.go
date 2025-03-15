package lib

import "time"

func WaitUntilTimestampChanges(currentTimestamp int64) (newTimestamp int64) {
	newTimestamp = time.Now().UTC().Unix()
	for currentTimestamp == newTimestamp {
		time.Sleep(time.Millisecond)
		newTimestamp = time.Now().UTC().Unix()
	}
	return
}
