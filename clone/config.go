package clone

import (
	"github.com/hashicorp/packer/template/interpolate"
	"github.com/jetbrains-infra/packer-builder-vsphere/common"
	"github.com/hashicorp/packer/packer"
)

type Config struct {
	common.BaseConfig `mapstructure:",squash"`
	CloneConfig       `mapstructure:",squash"`
	ctx               interpolate.Context
}

func NewConfig(raws ...interface{}) (*Config, []string, error) {
	c := new(Config)
	{
		err := common.DecodeConfig(c, &c.ctx, raws...)
		if err != nil {
			return nil, nil, err
		}
	}

	errs := new(packer.MultiError)
	errs = packer.MultiErrorAppend(errs, c.BaseConfig.Prepare(&c.ctx)...)
	errs = packer.MultiErrorAppend(errs, c.CloneConfig.Prepare()...)

	if len(errs.Errors) > 0 {
		return nil, nil, errs
	}

	return c, nil, nil
}
