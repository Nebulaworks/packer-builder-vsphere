package common

import (
	packerCommon "github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/template/interpolate"
)

type BaseConfig struct {
	packerCommon.PackerConfig `mapstructure:",squash"`
	ConnectConfig             `mapstructure:",squash"`
	HardwareConfig            `mapstructure:",squash"`
	Comm                      communicator.Config `mapstructure:",squash"`
	ShutdownConfig            `mapstructure:",squash"`
	CreateSnapshot            bool `mapstructure:"create_snapshot"`
	ConvertToTemplate         bool `mapstructure:"convert_to_template"`
}

func DecodeConfig(cfg interface{}, ctx *interpolate.Context, raws ...interface{}) error {
	err := config.Decode(cfg, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: ctx,
	}, raws...)
	return err
}

func (c *BaseConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error
	errs = append(errs, c.Comm.Prepare(ctx)...)
	errs = append(errs, c.ConnectConfig.Prepare()...)
	errs = append(errs, c.HardwareConfig.Prepare()...)
	errs = append(errs, c.ShutdownConfig.Prepare()...)

	return errs
}
