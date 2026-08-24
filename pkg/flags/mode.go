package flags

import (
	"context"

	"github.com/spf13/pflag"

	bqcachedclient "github.com/openshift/sippy/pkg/bigquery"
	"github.com/openshift/sippy/pkg/sippyserver"
	"github.com/openshift/sippy/pkg/testidentification"
)

type ModeFlags struct {
	Mode string
}

const (
	ModeACS  = "acs"
	ModeNone = "none"
)

func NewModeFlags() *ModeFlags {
	return &ModeFlags{
		Mode: ModeACS,
	}
}

func (f *ModeFlags) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&f.Mode, "mode", f.Mode, "Mode to use: {acs,none}")
}

func (f *ModeFlags) GetServerMode() sippyserver.Mode {
	if f.Mode == ModeACS {
		return sippyserver.ModeACS
	}

	return sippyserver.ModeKubernetes
}

func (f *ModeFlags) GetVariantManager(ctx context.Context, bqc *bqcachedclient.Client) testidentification.VariantManager {
	switch f.Mode {
	case ModeACS:
		mgr, err := testidentification.NewACSVariantManager(ctx, bqc)
		if err != nil {
			panic(err)
		}
		return mgr
	case ModeNone:
		return testidentification.NewEmptyVariantManager()
	default:
		panic("only acs or none is allowed")
	}
}
