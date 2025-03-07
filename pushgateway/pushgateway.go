package pushgateway

import (
	"bytes"
	"fmt"
	"github.com/XANi/collectd2metrics/datatypes"
	"go.uber.org/zap"
	"net/http"
	"net/url"
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
	cfg          Config
	l            *zap.SugaredLogger
	writeChannel chan datatypes.PrometheusWrite
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
		cfg:          cfg,
		l:            cfg.Logger,
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
	cl := http.Client{
		Timeout: p.cfg.Timeout,
	}
	for {
		events := []datatypes.PrometheusWrite{}
		deadline := time.After(p.cfg.MaxBatchDuration)
		var ev datatypes.PrometheusWrite
	tmout:
		for len(events) < p.cfg.MaxBatchLength {
			select {
			case ev = <-p.writeChannel:
				events = append(events, ev)
			case <-deadline:
				break tmout

			}
		}
		if len(events) > 0 {
			buf := bytes.Buffer{}
			for _, e := range events {
				e.Write(&buf)
			}
			resp, err := cl.Post(p.cfg.URL, "text/plain", &buf)
			if err != nil {
				p.l.Errorf("error sending request to %s: %s", p.cfg.URL, err)
				// TODO retry
				continue
			} else {
				resp.Body.Close()
			}
			if resp.StatusCode != 204 {
				p.l.Errorf("!240 status from url[%s]: [%d]%s", p.cfg.URL, resp.StatusCode, resp.Status)
			}
		}

	}
}
