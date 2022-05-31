package bandersnatchErrors

// keys for parameterName. We use constants so the IDE/compiler yells at us if we typo.

const PARTIAL_READ_FLAG = "PartialRead"
const IO_ERROR_FLAG = "IOError"

// Exported for cross-package testing. Will be removed/replaced by callback. Not part of the official interface
var GetDataPanicOnNonExistantKeys = false
