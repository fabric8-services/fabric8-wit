package ptr

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// Interface returns the pointer to the given interface.
func Interface(o interface{}) *interface{} { return &o }

// String returns the pointer to the given string.
func String(o string) *string { return &o }

// Time returns the pointer to the given time.Time.
func Time(o time.Time) *time.Time { return &o }

// UUID returns the pointer to the given uuid.UUID.
func UUID(o uuid.UUID) *uuid.UUID { return &o }

// ints ...

// Int returns the pointer to the given int.
func Int(o int) *int { return &o }

// Int8 returns the pointer to the given int8.
func Int8(o int8) *int8 { return &o }

// Int16 returns the pointer to the given int16.
func Int16(o int16) *int16 { return &o }

// Int32 returns the pointer to the given int32.
func Int32(o int32) *int32 { return &o }

// Int64 returns the pointer to the given int64.
func Int64(o int64) *int64 { return &o }

// uints ...

// Uint returns the pointer to the given uint.
func Uint(o uint) *uint { return &o }

// Uint8 returns the pointer to the given uint8.
func Uint8(o uint8) *uint8 { return &o }

// Uint16 returns the pointer to the given uint16.
func Uint16(o uint16) *uint16 { return &o }

// Uint32 returns the pointer to the given uint32.
func Uint32(o uint32) *uint32 { return &o }

// Uint64 returns the pointer to the given uint64.
func Uint64(o uint64) *uint64 { return &o }

// floats ...

// Float32 returns the pointer to the given float32.
func Float32(o float32) *float32 { return &o }

// Float64 returns the pointer to the given float32.
func Float64(o float64) *float64 { return &o }

// Bool returns the pointer to the given bool.
func Bool(o bool) *bool { return &o }
