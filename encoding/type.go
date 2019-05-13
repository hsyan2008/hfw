package encoding

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

//把数字也解析成字符串
type Str string

func (a *Str) UnmarshalJSON(data []byte) (err error) {
	s, err := strconv.Unquote(strings.Replace(string(data), "\\/", "/", -1))
	if err == nil {
		//如果本来是字符串
		*a = Str(s)
	} else {
		//如果本来就是数字
		*a = Str(data)
		err = nil
	}

	return
}

func (a *Str) MarshalJSON() (data []byte, err error) {

	return []byte(strconv.Quote(string(*a))), err
}

//把纯数字的字符串也解析成数字
type Int int

func (a *Int) UnmarshalJSON(data []byte) (err error) {
	s, err := strconv.Unquote(strings.Replace(string(data), "\\/", "/", -1))

	var as int
	if err == nil {
		//如果本来是字符串
		as, err = strconv.Atoi(s)
	} else {
		//如果本来就是数字
		as, err = strconv.Atoi(string(data))
	}

	if err != nil {
		return err
	}
	*a = Int(as)

	return
}

func (a *Int) MarshalJSON() (data []byte, err error) {

	return []byte(fmt.Sprintf("%d", *a)), err
}

const dateTimeFormart = "2006-01-02 15:04:05"

type DateTime time.Time

func (t *DateTime) UnmarshalJSON(data []byte) (err error) {
	now, err := time.ParseInLocation(`"`+dateTimeFormart+`"`, string(data), time.Local)
	*t = DateTime(now)
	return
}

func (t DateTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(dateTimeFormart)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, dateTimeFormart)
	b = append(b, '"')
	return b, nil
}
func (t DateTime) Unix() int64 {
	return time.Time(t).Unix()
}
func (t DateTime) UnixNano() int64 {
	return time.Time(t).UnixNano()
}

func (t DateTime) String() string {
	return time.Time(t).Format(dateTimeFormart)
}

const dateFormart = "2006-01-02"

type Date time.Time

func (t *Date) UnmarshalJSON(data []byte) (err error) {
	now, err := time.ParseInLocation(`"`+dateFormart+`"`, string(data), time.Local)
	*t = Date(now)
	return
}

func (t Date) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(dateFormart)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, dateFormart)
	b = append(b, '"')
	return b, nil
}

func (t Date) Unix() int64 {
	return time.Time(t).Unix()
}

func (t Date) UnixNano() int64 {
	return time.Time(t).UnixNano()
}

func (t Date) String() string {
	return time.Time(t).Format(dateFormart)
}
