package datatypes

type CollectdHTTPReq []CollectdHTTP

type CollectdHTTP struct {
	Values         []float64 `json:"values"`
	Dstypes        []string  `json:"dstypes"`
	Dsnames        []string  `json:"dsnames"`
	Time           float64   `json:"time"`
	Interval       float32   `json:"interval"`
	Host           string    `json:"host"`
	Plugin         string    `json:"plugin"`
	PluginInstance string    `json:"plugin_instance"`
	Type           string    `json:"type"`
	TypeInstance   string    `json:"type_instance"`
	Meta           struct {
		// data proxied from other node
		NetworkReceived bool `json:"network:received"`
		// other node address
		NetworkIPAddress string `json:"network:ip_address"`
	} `json:"meta"`
}
