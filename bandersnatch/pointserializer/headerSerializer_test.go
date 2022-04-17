package pointserializer

import "github.com/GottfriedHerold/Bandersnatch/internal/utils"

var _ headerDeserializer = &simpleHeaderDeserializer{}
var _ headerSerializer = &simpleHeaderSerializer{}
var _ headerDeserializer = &simpleHeaderSerializer{}

var _ utils.Clonable[*simpleHeaderDeserializer] = &simpleHeaderDeserializer{}
var _ utils.Clonable[*simpleHeaderSerializer] = &simpleHeaderSerializer{}
