package pointserializer

var _ headerDeserializer = &simpleHeaderDeserializer{}
var _ headerSerializer = &simpleHeaderSerializer{}
var _ headerDeserializer = &simpleHeaderSerializer{}
