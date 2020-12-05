package ptr_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/ptr"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type emptyTestStruct struct {
	someField int
}

func TestPtrInterfaceReturnValidPointers(t *testing.T) {
	expected := emptyTestStruct{5}
	actual := ptr.Interface(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrStringReturnValidPointers(t *testing.T) {
	expected := "str"
	actual := ptr.String(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrTimeReturnValidPointers(t *testing.T) {
	expected := time.Now()
	actual := ptr.Time(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrUUIDReturnValidPointers(t *testing.T) {
	expected := uuid.NewV4()
	actual := ptr.UUID(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrIntReturnValidPointers(t *testing.T) {
	expected := -42
	actual := ptr.Int(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrInt8ReturnValidPointers(t *testing.T) {
	expected := int8(127)
	actual := ptr.Int8(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrInt16ReturnValidPointers(t *testing.T) {
	expected := int16(32000)
	actual := ptr.Int16(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrInt32ReturnValidPointers(t *testing.T) {
	expected := int32(123)
	actual := ptr.Int32(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrInt64ReturnValidPointers(t *testing.T) {
	expected := int64(-287238)
	actual := ptr.Int64(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrUintReturnValidPointers(t *testing.T) {
	expected := uint(5)
	actual := ptr.Uint(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrUint8ReturnValidPointers(t *testing.T) {
	expected := uint8(0)
	actual := ptr.Uint8(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrUint16ReturnValidPointers(t *testing.T) {
	expected := uint16(887)
	actual := ptr.Uint16(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrUint32ReturnValidPointers(t *testing.T) {
	expected := uint32(2210)
	actual := ptr.Uint32(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrUint64ReturnValidPointers(t *testing.T) {
	expected := uint64(249842)
	actual := ptr.Uint64(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrFloat32ReturnValidPointers(t *testing.T) {
	expected := float32(-52.42)
	actual := ptr.Float32(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPtrFloat64ReturnValidPointers(t *testing.T) {
	expected := float64(1.24872)
	actual := ptr.Float64(expected)
	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}
