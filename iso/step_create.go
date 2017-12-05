package iso

import (
	"fmt"
	"github.com/hashicorp/packer/packer"
	"github.com/jetbrains-infra/packer-builder-vsphere/common"
	"github.com/jetbrains-infra/packer-builder-vsphere/driver"
	"github.com/mitchellh/multistep"
)

type CreateConfig struct {
	//ISODatastore        string `mapstructure:"iso_datastore"`
	//ISOPath             string `mapstructure:"iso_path"`
	VMName              string `mapstructure:"vm_name"`
	Folder              string `mapstructure:"folder"`
	Host                string `mapstructure:"host"`
	ResourcePool        string `mapstructure:"resource_pool"`
	Datastore           string `mapstructure:"datastore"`
	DiskThinProvisioned bool   `mapstructure:"disk_thin_provisioned"`
	DiskControlledType  string `mapstructure:"disk_controller_type"`
	GuestOS             string `mapstructure:"guest_os"`
}

func (c *CreateConfig) Prepare() []error {
	var errs []error

	if c.VMName == "" {
		errs = append(errs, fmt.Errorf("Target VM name is required"))
	}
	if c.Host == "" {
		errs = append(errs, fmt.Errorf("vSphere host is required"))
	}

	return errs
}

type StepCreateVM struct {
	hardwareConfig *common.HardwareConfig
	config         *CreateConfig
}

func (s *StepCreateVM) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	d := state.Get("driver").(*driver.Driver)

	ui.Say("Creating VM...")

	vm, err := d.CreateVM(&driver.CreateConfig{
		HardwareConfig: s.hardwareConfig.ToDriverConfig(),

		DiskThinProvisioned: s.config.DiskThinProvisioned,
		DiskControllerType:  s.config.DiskControlledType,
		Name:                s.config.VMName,
		Folder:              s.config.Folder,
		Host:                s.config.Host,
		ResourcePool:        s.config.ResourcePool,
		Datastore:           s.config.Datastore,
		GuestOS:             s.config.GuestOS,
	})

	if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}

	state.Put("vm", vm)
	return multistep.ActionContinue
}

func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if !cancelled && !halted {
		return
	}

	ui := state.Get("ui").(packer.Ui)

	st := state.Get("vm")
	if st == nil {
		return
	}
	vm := st.(*driver.VirtualMachine)

	ui.Say("Destroying VM...")

	err := vm.Destroy()
	if err != nil {
		ui.Error(err.Error())
	}
}
