package v1

const (
	// ConfigMapHashKey is the key of the annotation which is used by the Balancer.
	// Balancer wraps a Nginx instance, and the value corresponding to key ConfigMapHashKey is a hashing result.
	ConfigMapHashKey = "controller.exposer.xincechen.io/configmap-hash"
)
