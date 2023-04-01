package datatypes

type CollectdWriter interface {
	WriteCollectd(http CollectdHTTP)
}
