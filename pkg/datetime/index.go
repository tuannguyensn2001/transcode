package datetime

import "github.com/golang-module/carbon/v2"

var cb carbon.Carbon
var defaultTimeZone string = carbon.Bangkok

func init() {
	cb = carbon.SetTimezone(defaultTimeZone)
}

func New() carbon.Carbon {
	return cb
}

func Now() carbon.Carbon {
	return cb.Now()
}

func FromMilliSecond(sec int64) carbon.Carbon {
	return cb.CreateFromTimestampMilli(sec)
}
