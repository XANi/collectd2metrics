package datatypes

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type MetricType string

const (
	MetricTypeCounter        = MetricType("counter")
	MetricTypeGauge          = MetricType("gauge")
	MetricTypeHistogram      = MetricType("histogram")
	MetricTypeGaugeHistogram = MetricType("gaugehistogram")
	MetricTypeSummary        = MetricType("summary")
	MetricTypeInfo           = MetricType("info")
	MetricTypeStateset       = MetricType("stateset")
	MetricTypeUnknown        = MetricType("unknown")
)

type PrometheusWrite struct {
	Name   string
	Labels map[string]string
	Value  float64
	Type   MetricType
	TS     time.Time
}

func (p *PrometheusWrite) Write(f io.Writer) (n int, err error) {
	buf := bytes.Buffer{}
	buf.WriteString(promquotelabel(p.Name))
	if len(p.Labels) > 0 {
		tagList := []string{}
		for k, v := range p.Labels {
			tagList = append(tagList, promquotelabel(k)+"="+promquoteval(v))
		}
		sort.Strings(tagList)
		buf.WriteString("{" + strings.Join(tagList, ",") + "}")
	}
	buf.WriteString(fmt.Sprintf(" %f %d\n", p.Value, p.TS.UnixMilli()))
	return f.Write(buf.Bytes())
}
func promquoteval(s string) string {
	return fmt.Sprintf("%q", s)
}

var labelReplacer = strings.NewReplacer(
	" ", "_",
	"-", "_",
	"^", "_",
	"#", "_",
)

func promquotelabel(s string) string {
	return labelReplacer.Replace(s)
}
