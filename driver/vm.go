package driver

import (
	"errors"
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"time"
)

type VirtualMachine struct {
	vm     *object.VirtualMachine
	driver *Driver
}

type CloneConfig struct {
	Name         string
	Folder       string
	Host         string
	ResourcePool string
	Datastore    string
	LinkedClone  bool
}

type HardwareConfig struct {
	CPUs           int32
	CPUReservation int64
	CPULimit       int64
	RAM            int64
	RAMReservation int64
	RAMReserveAll  bool
}

type DiskConfig struct {
	DiskSizeKB      int64
	ThinProvisioned bool // ex: Thin Provision
	// TODO: more settings
}

type CreateConfig struct {
	HardwareConfig
	DiskConfig

	Annotation   string
	Name         string
	Folder       string
	Host         string
	ResourcePool string
	Datastore    string
	GuestOS      string // def: otherGuest
	Force        bool
}

func (d *Driver) NewVM(ref *types.ManagedObjectReference) *VirtualMachine {
	return &VirtualMachine{
		vm:     object.NewVirtualMachine(d.client.Client, *ref),
		driver: d,
	}
}

func (d *Driver) FindVM(name string) (*VirtualMachine, error) {
	vm, err := d.finder.VirtualMachine(d.ctx, name)
	if err != nil {
		return nil, err
	}
	return &VirtualMachine{
		vm:     vm,
		driver: d,
	}, nil
}

func (d *Driver) CreateVM(config *CreateConfig) (*VirtualMachine, error) {
	// See: vendor/github.com/vmware/govmomi/govc/vm/create.go

	createSpec := config.toConfigSpec()

	folder, err := d.FindFolder(config.Folder)
	if err != nil {
		return nil, err
	}

	resourcePool, err := d.FindResourcePool(config.Host, config.ResourcePool)
	if err != nil {
		return nil, err
	}

	host, err := d.FindHost(config.Host)
	if err != nil {
		return nil, err // TODO
	}

	datastore, err := getDatastoreOrDefault(d, host, config.Datastore)
	if err != nil {
		return nil, err // TODO
	}

	// Don't override existing file if parameter "Force" is not specified
	if !config.Force {
		vmxPath := fmt.Sprintf("%s/%s.vmx", config.Name, config.Name)
		if datastore.FileExists(vmxPath) {
			dsPath := datastore.Path(vmxPath)
			return nil, fmt.Errorf("File %s already exists", dsPath)
		}
	}

	devices := object.VirtualDeviceList{}

	if err = addDisk(&devices, config); err != nil {
		return nil, err // TODO
	}
	if err = addNetwork(&devices, config); err != nil {
		return nil, err // TODO
	}
	if err = addDrive(&devices, config); err != nil {
		return nil, err // TODO
	}

	createSpec.DeviceChange, err = devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err // TODO
	}

	/*
	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", datastore.Name()),
	}
	*/

	task, err := folder.folder.CreateVM(d.ctx, createSpec, resourcePool.pool, host.host)
	if err != nil {
		return nil, err // TODO
	}
	taskInfo, err := task.WaitForResult(d.ctx, nil)
	if err != nil {
		return nil, err
	}

	vmRef := taskInfo.Result.(types.ManagedObjectReference)
	return d.NewVM(&vmRef), nil
}

func (vm *VirtualMachine) Info(params ...string) (*mo.VirtualMachine, error) {
	var p []string
	if len(params) == 0 {
		p = []string{"*"}
	} else {
		p = params
	}
	var info mo.VirtualMachine
	err := vm.vm.Properties(vm.driver.ctx, vm.vm.Reference(), p, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (template *VirtualMachine) Clone(config *CloneConfig) (*VirtualMachine, error) {
	folder, err := template.driver.FindFolder(config.Folder)
	if err != nil {
		return nil, err
	}

	var relocateSpec types.VirtualMachineRelocateSpec

	pool, err := template.driver.FindResourcePool(config.Host, config.ResourcePool)
	if err != nil {
		return nil, err
	}
	poolRef := pool.pool.Reference()
	relocateSpec.Pool = &poolRef

	if config.Datastore == "" {
		host, err := template.driver.FindHost(config.Host)
		if err != nil {
			return nil, err
		}

		info, err := host.Info("datastore")
		if err != nil {
			return nil, err
		}

		if len(info.Datastore) > 1 {
			return nil, fmt.Errorf("Target host has several datastores. Specify 'datastore' parameter explicitly")
		}

		ref := info.Datastore[0].Reference()
		relocateSpec.Datastore = &ref
	} else {
		ds, err := template.driver.FindDatastore(config.Datastore)
		if err != nil {
			return nil, err
		}

		ref := ds.ds.Reference()
		relocateSpec.Datastore = &ref
	}

	var cloneSpec types.VirtualMachineCloneSpec
	cloneSpec.Location = relocateSpec
	cloneSpec.PowerOn = false

	if config.LinkedClone == true {
		cloneSpec.Location.DiskMoveType = "createNewChildDiskBacking"

		tpl, err := template.Info("snapshot")
		if err != nil {
			return nil, err
		}
		if tpl.Snapshot == nil {
			err = errors.New("`linked_clone=true`, but template has no snapshots")
			return nil, err
		}
		cloneSpec.Snapshot = tpl.Snapshot.CurrentSnapshot
	}

	task, err := template.vm.Clone(template.driver.ctx, folder.folder, config.Name, cloneSpec)
	if err != nil {
		return nil, err
	}

	info, err := task.WaitForResult(template.driver.ctx, nil)
	if err != nil {
		return nil, err
	}

	ref := info.Result.(types.ManagedObjectReference)
	vm := template.driver.NewVM(&ref)
	return vm, nil
}

func (vm *VirtualMachine) Destroy() error {
	task, err := vm.vm.Destroy(vm.driver.ctx)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(vm.driver.ctx, nil)
	return err
}

func (vm *VirtualMachine) Configure(config *HardwareConfig) error {
	confSpec := config.toConfigSpec()

	task, err := vm.vm.Reconfigure(vm.driver.ctx, confSpec)
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(vm.driver.ctx, nil)
	return err
}

func (vm *VirtualMachine) PowerOn() error {
	task, err := vm.vm.PowerOn(vm.driver.ctx)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(vm.driver.ctx, nil)
	return err
}

func (vm *VirtualMachine) WaitForIP() (string, error) {
	ip, err := vm.vm.WaitForIP(vm.driver.ctx)
	if err != nil {
		return "", err
	}
	return ip, nil
}

func (vm *VirtualMachine) PowerOff() error {
	state, err := vm.vm.PowerState(vm.driver.ctx)
	if err != nil {
		return err
	}

	if state == types.VirtualMachinePowerStatePoweredOff {
		return nil
	}

	task, err := vm.vm.PowerOff(vm.driver.ctx)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(vm.driver.ctx, nil)
	return err
}

func (vm *VirtualMachine) StartShutdown() error {
	err := vm.vm.ShutdownGuest(vm.driver.ctx)
	return err
}

func (vm *VirtualMachine) WaitForShutdown(timeout time.Duration) error {
	shutdownTimer := time.After(timeout)
	for {
		powerState, err := vm.vm.PowerState(vm.driver.ctx)
		if err != nil {
			return err
		}
		if powerState == "poweredOff" {
			break
		}

		select {
		case <-shutdownTimer:
			err := errors.New("Timeout while waiting for machine to shut down.")
			return err
		default:
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func (vm *VirtualMachine) CreateSnapshot(name string) error {
	task, err := vm.vm.CreateSnapshot(vm.driver.ctx, name, "", false, false)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(vm.driver.ctx, nil)
	return err
}

func (vm *VirtualMachine) ConvertToTemplate() error {
	err := vm.vm.MarkAsTemplate(vm.driver.ctx)
	return err
}

func (config HardwareConfig) toConfigSpec() types.VirtualMachineConfigSpec {
	var confSpec types.VirtualMachineConfigSpec

	confSpec.NumCPUs = config.CPUs
	confSpec.MemoryMB = config.RAM

	var cpuSpec types.ResourceAllocationInfo
	cpuSpec.Reservation = config.CPUReservation
	cpuSpec.Limit = config.CPULimit
	confSpec.CpuAllocation = &cpuSpec

	var ramSpec types.ResourceAllocationInfo
	ramSpec.Reservation = config.RAMReservation
	confSpec.MemoryAllocation = &ramSpec

	return confSpec
}

func (config CreateConfig) toConfigSpec() types.VirtualMachineConfigSpec {
	confSpec := config.HardwareConfig.toConfigSpec()
	confSpec.Name = config.Name
	confSpec.Annotation = config.Annotation
	confSpec.GuestId = config.GuestOS
	return confSpec
}

func getDatastoreOrDefault(d *Driver, host *Host, name string) (*Datastore, error) {
	// TODO
	/*if config.Datastore == "" {
		host, err := template.driver.FindHost(config.Host)
		if err != nil {
			return nil, err
		}

		info, err := host.Info("datastore")
		if err != nil {
			return nil, err
		}

		if len(info.Datastore) > 1 {
			return nil, fmt.Errorf("Target host has several datastores. Specify 'datastore' parameter explicitly")
		}

		ref := info.Datastore[0].Reference()
		relocateSpec.Datastore = &ref
	} else {
		ds, err := template.driver.FindDatastore(config.Datastore)
		if err != nil {
			return nil, err
		}

		ref := ds.ds.Reference()
		relocateSpec.Datastore = &ref
	}*/
	return nil, fmt.Errorf("not implemented")
}

func addDisk(devices *object.VirtualDeviceList, config *CreateConfig) error {
	// FIXME: controller type should be customizable
	device, err := devices.CreateSCSIController("scsi")
	if err != nil {
		return err
	}
	*devices = append(*devices, device)
	controller, err := devices.FindDiskController(devices.Name(device))
	if err != nil {
		return err
	}

	if config.DiskSizeKB == 0 {
		return fmt.Errorf("zero sized disk not supported") // FIXME
	}

	disk := &types.VirtualDisk {
		VirtualDevice: types.VirtualDevice{
			Key: devices.NewKey(),
			Backing: &types.VirtualDiskFlatVer2BackingInfo{
				DiskMode:        string(types.VirtualDiskModePersistent), // FIXME: should be customizable?
				ThinProvisioned: types.NewBool(config.ThinProvisioned),
			},
		},
		CapacityInKB: config.DiskSizeKB,
	}

	devices.AssignController(disk, controller)
	*devices = append(*devices, disk)

	return nil
}

func addNetwork(devices *object.VirtualDeviceList, config *CreateConfig) error {
	// TODO
	return fmt.Errorf("not implemented")
}

func addDrive(devices *object.VirtualDeviceList, config *CreateConfig) error {
	// TODO
	return fmt.Errorf("not implemented")
}
