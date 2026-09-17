package marketplace

import "errors"

var (
	ErrNoSourceMapping = errors.New("no source mapping found for product")
	ErrProviderFetch   = errors.New("provider fetch failed")
)

const (
	warningDeferredNoAPI = "Bu kaynak için canlı API erişimi yapılandırılmamış."
	warningCooldown      = "Ürün kısa süre önce canlı kaynaktan güncellendi."
	warningReviewPersist = "Ürün bilgileri güncellendi ancak yorumlar güncellenemedi."
)
