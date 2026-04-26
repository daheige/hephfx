package logger

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestFieldToValue(t *testing.T) {
	z := zap.String("foo", "bar")
	v := FieldToValue(z)
	log.Printf("%v", v)
	assert.Equal(t, z.String, v)

	m := zap.Any("foo", map[string]interface{}{
		"foo": "bar",
	})
	v2 := FieldToValue(m)
	log.Printf("v2:%v", v2)
	assert.Equal(t, m.Interface, v2)

	fmt.Println(MaskString("34555555555555555555555"))
	fmt.Println(MaskString("34555555555555555555555gggggggggg66666666666"))
}
