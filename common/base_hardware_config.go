package common

import (
	"errors"
	"github.com/jetbrains-infra/packer-builder-vsphere/driver"
)

type BaseHardwareConfig struct {
	CPUs           int32 `mapstructure:"CPUs"`
	CPUReservation int64 `mapstructure:"CPU_reservation"`
	CPULimit       int64 `mapstructure:"CPU_limit"`
	RAM            int64 `mapstructure:"RAM"`
	RAMReservation int64 `mapstructure:"RAM_reservation"`
	RAMReserveAll  bool  `mapstructure:"RAM_reserve_all"`
	DiskSize       int64 `mapstructure:"disk_size"`
}

func (c *BaseHardwareConfig) Prepare() []error {
	var errs []error

	if c.RAMReservation > 0 && c.RAMReserveAll {
		errs = append(errs, errors.New("'RAM_reservation' and 'RAM_reserve_all' cannot be used together"))
	}

	return errs
}

func (c *BaseHardwareConfig) ToDriverHardwareConfig() driver.HardwareConfig {
	return driver.HardwareConfig{
		CPUs:           c.CPUs,
		CPUReservation: c.CPUReservation,
		CPULimit:       c.CPULimit,
		RAM:            c.RAM,
		RAMReservation: c.RAMReservation,
		RAMReserveAll:  c.RAMReserveAll,
		DiskSize:       c.DiskSize,
	}
}
