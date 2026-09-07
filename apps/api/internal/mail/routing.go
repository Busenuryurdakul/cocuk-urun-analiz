package mail

import "strings"

// ResolveDeliveryAddress maps test-only account emails to a reachable inbox.
// The account email stays unchanged in auth; only outbound delivery is redirected.
func ResolveDeliveryAddress(to string) string {
	switch strings.TrimSpace(strings.ToLower(to)) {
	case "yurdakulbusenur.38test@gmail.com":
		return "yurdakulbusenur.38@gmail.com"
	default:
		return strings.TrimSpace(to)
	}
}

func normalizeMessage(msg Message) Message {
	msg.To = ResolveDeliveryAddress(msg.To)
	return msg
}
