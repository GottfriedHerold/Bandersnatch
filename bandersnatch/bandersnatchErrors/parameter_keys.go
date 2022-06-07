package bandersnatchErrors

// keys for parameterNames. We use constants so the IDE/compiler yells at us if we typo.
// Note that the strings should be capitalized, in order to use the alternative interface of errorsWithData.

const PARTIAL_READ_FLAG = "PartialRead"
const IO_ERROR_FLAG = "IOError"
