package clone

import (
	"github.com/mitchellh/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/jetbrains-infra/packer-builder-vsphere/driver"
	"github.com/jetbrains-infra/packer-builder-vsphere/common"
)

type HardwareConfig struct {
	common.BaseHardwareConfig
}

func (c *HardwareConfig) Prepare() []error {
	return c.BaseHardwareConfig.Prepare()
}


type StepConfigureHardware struct {
	Config *HardwareConfig
}

func (s *StepConfigureHardware) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	vm := state.Get("vm").(*driver.VirtualMachine)

	if *s.Config != (HardwareConfig{}) {
		ui.Say("Customizing hardware parameters...")

		driverConfig := s.Config.ToDriverHardwareConfig()
		err := vm.Configure(&driverConfig)
		if err != nil {
			state.Put("error", err)
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

func (s *StepConfigureHardware) Cleanup(multistep.StateBag) {}
