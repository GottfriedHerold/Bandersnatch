package bandersnatchErrors

// This file contains keys for parameterNames for the error with parameters/data framework.
// We use constants so the IDE/compiler yells at us if we typo.
// Note that the strings should be capitalized, in order to use the alternative interface of errorsWithData.

// TODO: Move this file?

// const PARTIAL_READ_FLAG = "PartialRead"
// const IO_ERROR_FLAG = "IOError"

const PARTIAL_WRITE = "PartialWrite"
const PARTIAL_READ = "PartialRead"

const BYTES_READ = "BytesRead"
const BYTES_WRITTEN = "BytesWritten"
