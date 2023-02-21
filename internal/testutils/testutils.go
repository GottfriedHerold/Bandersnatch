// This contains cross-package functions that are used in tests for multiple packages.
// We don't want to export those to users, so they are in an internal package.
// NOTE: We only put function here that don't import anything outside the standard library to avoid cyclic dependencies.
// (internal/utils is OK)
package testutils
