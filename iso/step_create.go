package iso

import (
	"errors"
	"fmt"
	"github.com/hashicorp/packer/packer"
	"github.com/jetbrains-infra/packer-builder-vsphere/common"
	"github.com/jetbrains-infra/packer-builder-vsphere/driver"
	"github.com/mitchellh/multistep"
)

type HardwareConfig struct {
	common.BaseHardwareConfig `mapstructure:",squash"`
	DiskThinProvisioned       bool   `mapstructure:"disk_thin_provisioned"`
	DiskControlledType        string `mapstructure:"disk_controller_type"`
}

func (c *HardwareConfig) Prepare() []error {
	var errs []error

	errs = append(errs, c.BaseHardwareConfig.Prepare()...)

	if c.DiskSize <= 0 {
		errs = append(errs, errors.New("Disk size is required"))
	}

	return errs
}

type CreateConfig struct {
	HardwareConfig `mapstructure:",squash"`

	VMName       string `mapstructure:"vm_name"`
	Folder       string `mapstructure:"folder"`
	Host         string `mapstructure:"host"`
	ResourcePool string `mapstructure:"resource_pool"`
	Datastore    string `mapstructure:"datastore"`
	GuestOSType  string `mapstructure:"guest_os_type"`
}

func (c *CreateConfig) Prepare() []error {
	var errs []error

	// do recursive calls
	errs = append(errs, c.HardwareConfig.Prepare()...)

	// check for errors
	if c.VMName == "" {
		errs = append(errs, fmt.Errorf("Target VM name is required"))
	}
	if c.Host == "" {
		errs = append(errs, fmt.Errorf("vSphere host is required"))
	}

	if len(errs) > 0 {
		return errs
	}

	// set default values
	if c.GuestOSType == "" {
		c.GuestOSType = "otherGuest"
	}

	return []error{}
}

type StepCreateVM struct {
	config *CreateConfig
}

func (s *StepCreateVM) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	d := state.Get("driver").(*driver.Driver)

	ui.Say("Creating VM...")

	vm, err := d.CreateVM(&driver.CreateConfig{
		HardwareConfig: s.config.HardwareConfig.ToDriverHardwareConfig(),

		DiskThinProvisioned: s.config.DiskThinProvisioned,
		DiskControllerType:  s.config.DiskControlledType,
		Name:                s.config.VMName,
		Folder:              s.config.Folder,
		Host:                s.config.Host,
		ResourcePool:        s.config.ResourcePool,
		Datastore:           s.config.Datastore,
		GuestOS:             s.config.GuestOSType,
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
