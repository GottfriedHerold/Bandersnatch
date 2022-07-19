package pointserializer

type testMultiSerializer = multiSerializer[pointSerializerXY, *pointSerializerXY]

var _ CurvePointSerializerModifyable = &multiSerializer[pointSerializerXY, *pointSerializerXY]{}
