# collectd2metrics


Takes data from collectd's HTTP JSON protocol and sends it to other monitoring solutions


example input:
```json
[
  {
    "values": [
      11.5267947421638
    ],
    "dstypes": [
      "gauge"
    ],
    "dsnames": [
      "value"
    ],
    "time": 1680363662.935,
    "interval": 10,
    "host": "example.com",
    "plugin": "cpu",
    "plugin_instance": "1",
    "type": "percent",
    "type_instance": "user",
    "meta": {
      "network:received": true,
      "network:ip_address": "10.0.0.2"
    }
  },
  {
    "values": [
      0.504032258064516
    ],
    "dstypes": [
      "gauge"
    ],
    "dsnames": [
      "value"
    ],
    "time": 1680363662.935,
    "interval": 10,
    "host": "example.com",
    "plugin": "cpu",
    "plugin_instance": "0",
    "type": "percent",
    "type_instance": "nice",
    "meta": {
      "network:received": true,
      "network:ip_address": "10.0.0.2"
    }
  }
]
```


### VictoriaMetrics import format:

``` json
{
  "metric": {
    "__name__": "vm_request_errors_total",
    "service": "vmselect",
    "type": "rpcClient",
    "host": "cthulhu",
    "name": "vmselect",
    "action": "labelNames",
    "addr": "127.0.0.1:8401"
  },
  "values": [
    0,
  ],
   "timestamps": [
    1680362450860,
    1680362460862,
    1680362479035,
    1680362480859,
],
}
```

