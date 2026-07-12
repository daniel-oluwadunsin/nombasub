package nomba

import (
	"log"
	"os"
)

// debugLoggingEnabled reports whether verbose Nomba request/response logging is
// turned on. It is off by default so full payloads — which contain customer PII
// and payment/mandate details — are never written to logs in production. Set
// NOMBA_DEBUG_LOGGING=true to enable while diagnosing integration issues.
func debugLoggingEnabled() bool {
	return os.Getenv("NOMBA_DEBUG_LOGGING") == "true"
}

// debugLog logs only when verbose Nomba logging is enabled.
func debugLog(format string, args ...any) {
	if debugLoggingEnabled() {
		log.Printf(format, args...)
	}
}
