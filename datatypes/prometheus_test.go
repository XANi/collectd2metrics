package datatypes

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPrometheusWrite_Write(t *testing.T) {
	v := PrometheusWrite{
		Name: "val",
		Labels: map[string]string{
			"host-name": "example.com",
			"long key":  "value",
			"example":   "funky\n\"label\" \\value",
		},
		Value: 12.45678912345657,
		TS:    time.UnixMilli(1680362104124),
	}
	var buf bytes.Buffer
	v.Write(&buf)
	assert.Equal(t, `val{example="funky\n\"label\" \\value",host_name="example.com",long_key="value"} 12.456789 1680362104124`+"\n", buf.String())
}

func BenchmarkPrometheusWrite_Write(b *testing.B) {
	v := PrometheusWrite{
		Name: "val",
		Labels: map[string]string{
			"host-name": "example.com",
			"key":       "value",
			"example":   "test",
		},
		Value: 12.45678912345657,
		TS:    time.UnixMilli(1680362104124),
	}
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		v.Write(&buf)
	}
}
