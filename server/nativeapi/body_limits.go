package nativeapi

// Request body limits for JSON endpoints, so an authenticated client cannot make the
// server buffer an unbounded body. Backported from upstream navidrome #6506.
const (
	// maxSmallJSONBody fits small payloads such as library selections or chat messages.
	maxSmallJSONBody = 64 << 10
	// maxNativeJSONBody leaves room for large track-id lists (queues, playlist adds).
	maxNativeJSONBody = 4 << 20
	// maxPluginUpdateBody leaves room for large plugin configs.
	maxPluginUpdateBody = 1 << 20
)
