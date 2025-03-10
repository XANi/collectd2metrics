package promwriter

import (
	"bytes"
	"fmt"
	"github.com/XANi/collectd2metrics/datatypes"
	"github.com/efigence/go-mon"
	"github.com/golang/protobuf/proto"
	"github.com/klauspost/compress/snappy"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Config struct {
	URL              string
	Timeout          time.Duration      `yaml:"timeout"`
	MaxBatchDuration time.Duration      `yaml:"max_batch_duration"`
	MaxBatchLength   int                `yaml:"max_batch_length"`
	HostLabelFile    string             `yaml:"host_label_file"`
	Logger           *zap.SugaredLogger `yaml:"-"`
}

type PromWriter struct {
	cfg             Config
	l               *zap.SugaredLogger
	writeChannel    chan datatypes.PrometheusWrite
	http            *http.Client
	monEvCount      mon.Metric
	monReqOkCount   mon.Metric
	monReqFailCount mon.Metric
}

func New(cfg Config) (*PromWriter, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = time.Second * 10
	}
	if cfg.MaxBatchDuration <= 0 {
		cfg.MaxBatchDuration = time.Second * 3
	}
	if cfg.MaxBatchLength <= 0 {
		cfg.MaxBatchLength = 1000
	}
	w := PromWriter{
		cfg:             cfg,
		l:               cfg.Logger,
		monEvCount:      mon.GlobalRegistry.MustRegister("promwriter_events_total", mon.NewCounter()),
		monReqOkCount:   mon.GlobalRegistry.MustRegister("promwriter_requests_total", mon.NewCounter(), map[string]string{"state": "ok"}),
		monReqFailCount: mon.GlobalRegistry.MustRegister("promwriter_requests_total", mon.NewCounter(), map[string]string{"state": "fail"}),
		http: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       cfg.Timeout,
		},
		writeChannel: make(chan datatypes.PrometheusWrite, cfg.MaxBatchLength*2),
	}
	url, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing prometheus URL: %w", err)
	} else {
		// cut down url so any basic auth pass won't show in logs
		cfg.Logger.Infof("starting prometheus writer to %s%s", url.Host, url.Path)
	}
	if len(cfg.HostLabelFile) > 0 {
		w.l.Infof("loading host label map file %s", cfg.HostLabelFile)
		err := w.loadHostLabels(cfg.HostLabelFile)
		if err != nil {
			return nil, err
		}
		go w.updateHostLabels()
	}
	go w.writer()
	return &w, nil
}

func (p *PromWriter) WriteChannel() chan datatypes.PrometheusWrite {
	return p.writeChannel
}

func (p *PromWriter) WriteCollectd(c datatypes.CollectdHTTP) {
	prom := datatypes.PrometheusWrite{
		TS: time.UnixMilli(int64(c.Time * 1000)),
		Labels: map[string]string{
			"host": c.Host,
		},
	}
	switch c.Type {
	case "gauge":
		prom.Name = c.Plugin
	case "counter":
		prom.Name = c.Plugin + "_total"
	case "derive":
		prom.Name = c.Plugin + "_total"
	case prom.Name: // if type has same name as plugin, dont repeat
	default:
		prom.Name = c.Plugin + "_" + strings.TrimLeft(strings.TrimPrefix(c.Type, c.Plugin), "_-")
		prom.Name = strings.Trim(prom.Name, "-_")
	}
	if len(c.PluginInstance) > 0 {
		prom.Labels["instance"] = c.PluginInstance
	}
	if len(c.TypeInstance) > 0 {
		prom.Labels["type_instance"] = c.TypeInstance
	}
	maxWriteDelay := time.After(time.Second * 10)
	if len(c.Values) == 1 {
		prom.Value = c.Values[0]
		select {
		case p.writeChannel <- prom:
		case <-maxWriteDelay:
			p.l.Warnf("queue delay exceeded")
		}
	} else {
		for idx, v := range c.Values {
			promEv := prom
			promEv.Labels = map[string]string{}
			for k, v := range prom.Labels {
				promEv.Labels[k] = v
			}
			promEv.Labels["type"] = c.Dsnames[idx]
			switch c.Dstypes[idx] {
			case "derive", "counter":
				promEv.Type = datatypes.MetricTypeCounter
			case "gauge":
				promEv.Type = datatypes.MetricTypeGauge
			}
			promEv.Value = v
			select {
			case p.writeChannel <- promEv:
			case <-maxWriteDelay:
				p.l.Warnf("queue delay exceeded")
			}
		}
	}

}

func (p *PromWriter) writer() {

	for {
		events := []datatypes.PrometheusWrite{}
		deadline := time.After(p.cfg.MaxBatchDuration)
		var prWr datatypes.PrometheusWrite
	tmout:
		for len(events) < p.cfg.MaxBatchLength {
			select {
			case prWr = <-p.writeChannel:
				events = append(events, prWr)
			case <-deadline:
				break tmout
			}
		}
		if len(events) > 0 {
			wr := &prompb.WriteRequest{
				Timeseries: make([]prompb.TimeSeries, 0),
			}

			for _, e := range events {
				dp := prompb.TimeSeries{
					Labels: []prompb.Label{{
						Name:  "__name__",
						Value: e.Name,
					}},
				}
				for k, v := range e.Labels {
					dp.Labels = append(dp.Labels, prompb.Label{
						Name:  k,
						Value: v,
					})
				}
				// protocol requires them to be sorted, just in case some server is stupid enough to enforce this silliness
				sort.Slice(dp.Labels, func(i, j int) bool {
					return dp.Labels[i].Name < dp.Labels[j].Name
				})
				dp.Samples = []prompb.Sample{{
					Timestamp: e.TS.UnixMilli(),
					Value:     e.Value,
				}}

				wr.Timeseries = append(wr.Timeseries, dp)
				//	wr.Metadata = append(wr.Metadata, prompb.MetricMetadata{
				//		Type:             0,
				//		MetricFamilyName: "",
				//		Help:             "",
				//		Unit:             "",
				//	})
			}
			p.monEvCount.Update(float64(len(events)))
			b, err := proto.Marshal(wr)
			buf := snappy.Encode(nil, b)
			req, err := http.NewRequest("POST", p.cfg.URL, bytes.NewBuffer(buf))

			req.Header.Set("Content-Encoding", "snappy")
			req.Header.Set("Content-Type", "application/x-protobuf")
			req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

			resp, err := p.http.Do(req)
			if err != nil {
				p.l.Errorf("error sending request to %s: %s", p.cfg.URL, err)
				// TODO retry
				p.monReqFailCount.Update(1)
				continue
			} else {
				resp.Body.Close()
			}
			if resp.StatusCode != 204 {
				p.l.Errorf("!240 status from url[%s]: [%d]%s", p.cfg.URL, resp.StatusCode, resp.Status)
				p.monReqFailCount.Update(1)
			} else {
				p.monReqOkCount.Update(1)
			}
		}

	}
}
