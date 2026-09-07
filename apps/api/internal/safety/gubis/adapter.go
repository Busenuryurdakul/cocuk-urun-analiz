package gubis

import (
	"context"
	"errors"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

const OfficialPortal = "https://gubis.ticaret.gov.tr/"

var (
	ErrNoStablePublicAPI = errors.New("gubis has no stable public machine-readable API")
	AllowedHosts         = map[string]bool{
		"gubis.ticaret.gov.tr":        true,
		"www.gubis.ticaret.gov.tr":    true,
		"guvensizurun.ticaret.gov.tr": true,
	}
)

type Adapter struct{}

func NewAdapter() *Adapter { return &Adapter{} }

func (a *Adapter) Source() string { return recall.SourceGUBIS }

func (a *Adapter) SearchRecalls(ctx context.Context, _ recall.Identity) ([]recall.Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := safety.ValidateOfficialURL(OfficialPortal, AllowedHosts); err != nil {
		return nil, err
	}
	return nil, ErrNoStablePublicAPI
}
