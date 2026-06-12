package export

// DataKind 导出数据类别
type DataKind int8

const (
	_         = DataKind(iota)
	DataJson  // json
	DataBytes // bytes
	DataBson  // bson from mongo.
	dataKindMax
)

// Valid 检查是否是有效的数据类别
func (dk DataKind) Valid() bool {
	return dk > 0 && dk < dataKindMax
}

var dataKindStrings = [...]string{
	DataJson:  "json",
	DataBytes: "bytes",
	DataBson:  "bson",
}

// String 转换为字符串
func (dk DataKind) String() string {
	return dataKindStrings[dk]
}

var dataStringKinds = map[string]DataKind{
	DataJson.String():  DataJson,
	DataBytes.String(): DataBytes,
	DataBson.String():  DataBson,
}

// FromString 从字符串转换
func (dk *DataKind) FromString(s string) bool {
	if k, ok := dataStringKinds[s]; ok {
		*dk = k
		return true
	}
	return false
}
