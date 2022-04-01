package pointserializer

/*
import (
	. "github.com/GottfriedHerold/Bandersnatch/bandersnatch"
)
*/

/*
type headerSerializer struct {
	// headerAll           []byte
	headerPerCurvePoint []byte
	// footer              []byte
	// headerAllReader     func(input io.Reader) (bytes_read int, err error, curvePointsToRead int, extra interface{})
	// headerPointReader   func(input io.Reader) (bytes_read int, err error, extra interface{})
}
*/

/*
func (hs *headerSerializer) clone() (ret headerSerializer) {
	if hs.headerPerCurvePoint == nil {
		ret.headerPerCurvePoint = nil
	} else {
		ret.headerPerCurvePoint = make([]byte, len(hs.headerPerCurvePoint))
		copy(ret.headerPerCurvePoint, hs.headerPerCurvePoint)
	}
	return
}
*/

/*
type simpleDeserializer struct {
	headerSerializer
	pointSerializer pointSerializerInterface
}

func (s *simpleDeserializer) Deserialize(outputPoint CurvePointPtrInterfaceWrite, inputStream io.Reader, trustLevel IsPointTrusted) (bytesRead int, err error) {
	var bytesJustRead int
	if s.headerSerializer.headerPerCurvePoint != nil {
		bytesRead, err = consumeExpectRead(inputStream, s.headerSerializer.headerPerCurvePoint)
		if err != nil {
			return
		}
	}
	bytesJustRead, err = s.pointSerializer.deserializeCurvePoint(inputStream, outputPoint, trustLevel)
	bytesRead += bytesJustRead
	return
}

func (s *simpleDeserializer) Serialize(inputPoint CurvePointPtrInterfaceRead, outputStream io.Writer) (bytesWritten int, err error) {
	var bytesJustWritten int
	if s.headerSerializer.headerPerCurvePoint != nil {
		bytesWritten, err = outputStream.Write(s.headerSerializer.headerPerCurvePoint)
		if err != nil {
			return
		}
	}
	bytesJustWritten, err = s.pointSerializer.serializeCurvePoint(outputStream, inputPoint)
	bytesWritten += bytesJustWritten
	return
}

func (s *simpleDeserializer) Clone() (ret simpleDeserializer) {
	ret.headerSerializer = s.headerSerializer.clone()
	ret.pointSerializer = s.pointSerializer.clone()
	return
}

func (s *simpleDeserializer) WithEndianness(e binary.ByteOrder) (ret simpleDeserializer) {
	ret = s.Clone()
	ret.pointSerializer.setEndianness(e)
	return
}

func (s *simpleDeserializer) WithHeader(perPointHeader []byte) (ret simpleDeserializer) {
	ret = s.Clone()
	if perPointHeader == nil {
		s.headerPerCurvePoint = nil
	} else {
		s.headerPerCurvePoint = make([]byte, len(perPointHeader))
		copy(s.headerPerCurvePoint, perPointHeader)
	}
	return
}

*/
