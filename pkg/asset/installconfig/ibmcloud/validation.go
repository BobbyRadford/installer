package ibmcloud

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/ibmcloud"
)

// Validate executes platform-specific validation.
func Validate(client API, ic *types.InstallConfig) error {
	allErrs := field.ErrorList{}
	childPath := field.NewPath("platorm").Child("ibmcloud")
	allErrs = append(allErrs, validatePlatform(client, ic, childPath)...)

	if ic.ControlPlane != nil && ic.ControlPlane.Platform.IBMCloud != nil {
		machinePool := ic.ControlPlane.Platform.IBMCloud
		fldPath := field.NewPath("controlPlane").Child("platform").Child("ibmcloud")
		allErrs = append(allErrs, validateMachinePool(client, ic.IBMCloud, machinePool, fldPath)...)
	}
	for idx, compute := range ic.Compute {
		machinePool := compute.Platform.IBMCloud
		fldPath := field.NewPath("compute").Index(idx).Child("platform").Child("ibmcloud")
		if machinePool != nil {
			allErrs = append(allErrs, validateMachinePool(client, ic.IBMCloud, machinePool, fldPath)...)
		}
	}

	return allErrs.ToAggregate()
}

func validatePlatform(client API, ic *types.InstallConfig, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateRegion(client, ic.Platform.IBMCloud.Region, path)...)
	allErrs = append(allErrs, validateCISInstanceCRN(client, ic.BaseDomain, ic.Platform.IBMCloud, path)...)
	allErrs = append(allErrs, validateClusterOSImage(client, ic.Platform.IBMCloud.ClusterOSImage, ic.Platform.IBMCloud.Region, path)...)

	if ic.Platform.IBMCloud.ResourceGroup != "" {
		allErrs = append(allErrs, validateResourceGroup(client, ic, path)...)
	}

	if ic.Platform.IBMCloud.VPC != "" || len(ic.Platform.IBMCloud.Subnets) > 0 {
		allErrs = append(allErrs, validateNetworking(client, ic, path)...)
	}

	if ic.Platform.IBMCloud.DefaultMachinePlatform != nil {
		allErrs = append(allErrs, validateMachinePool(client, ic.IBMCloud, ic.Platform.IBMCloud.DefaultMachinePlatform, path)...)
	}
	return allErrs
}

func validateMachinePool(client API, platform *ibmcloud.Platform, machinePool *ibmcloud.MachinePool, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if machinePool.Type != "" {
		allErrs = append(allErrs, validateMachinePoolType(client, machinePool.Type, path)...)
	}

	if len(machinePool.Zones) > 0 {
		allErrs = append(allErrs, validateMachinePoolZones(client, platform.Region, machinePool.Zones, path)...)
	}

	if machinePool.BootVolume != nil {
		allErrs = append(allErrs, validateMachinePoolBootVolume(client, *machinePool.BootVolume, path)...)
	}

	return allErrs
}

func validateMachinePoolType(client API, machineType string, path *field.Path) field.ErrorList {
	vsiProfiles, err := client.GetVSIProfiles(context.TODO())
	if err != nil {
		return field.ErrorList{field.InternalError(path.Child("type"), err)}
	}

	for _, profile := range vsiProfiles {
		if *profile.Name == machineType {
			return field.ErrorList{}
		}
	}

	return field.ErrorList{field.NotFound(path.Child("type"), machineType)}
}

func validateMachinePoolZones(client API, region string, zones []string, path *field.Path) field.ErrorList {
	regionalZones, err := client.GetVPCZonesForRegion(context.TODO(), region)
	if err != nil {
		return field.ErrorList{field.InternalError(path.Child("zones"), err)}
	}

	for idx, zone := range zones {
		if !contains(regionalZones, zone) {
			return field.ErrorList{field.Invalid(path.Child("zones").Index(idx), zone, fmt.Sprintf("zone : %s not found in region : %s", zone, region))}
		}
	}
	return field.ErrorList{}
}

func validateMachinePoolBootVolume(client API, bootVolume ibmcloud.BootVolume, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Make sure the encryptionKey exists
	key, err := client.GetEncryptionKey(context.TODO(), bootVolume.EncryptionKey)
	if err != nil {
		return field.ErrorList{field.InternalError(path.Child("bootVolume").Child("encryptionKey"), err)}
	}

	if key == nil {
		return field.ErrorList{field.NotFound(path.Child("bootVolume").Child("encryptionKey"), bootVolume.EncryptionKey)}
	}

	return allErrs
}

func validateRegion(client API, region string, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO: IBM: Region validation already happens in
	// pkg/types/ibmcloud/validation/platform.go. Do we need it here too?
	return allErrs
}

func validateResourceGroup(client API, ic *types.InstallConfig, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if ic.IBMCloud.ResourceGroup != "" {
		resourceGroups, err := client.GetResourceGroups(context.TODO())
		if err != nil {
			return append(allErrs, field.InternalError(path.Child("resourceGroup"), err))
		}

		found := false
		for _, rg := range resourceGroups {
			if rg.ID == ic.IBMCloud.ResourceGroup || rg.Name == ic.IBMCloud.ResourceGroup {
				found = true
			}
		}

		if !found {
			return append(allErrs, field.NotFound(path.Child("resourceGroup"), ic.IBMCloud.ResourceGroup))
		}
	}

	return allErrs
}

func validateCISInstanceCRN(client API, baseDomain string, platform *ibmcloud.Platform, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := client.GetCISInstance(context.TODO(), platform.CISInstanceCRN); err != nil {
		allErrs = append(allErrs, field.NotFound(path.Child("cisInstanceCRN"), platform.CISInstanceCRN))
	} else {
		id, err := client.GetZoneIDByName(context.TODO(), platform.CISInstanceCRN, baseDomain)
		if err != nil || id == "" {
			details := fmt.Sprintf("the cis instance does not have an active DNS zone for the base domain: %s", baseDomain)
			allErrs = append(allErrs, field.Invalid(path.Child("cisInstanceCRN"), platform.CISInstanceCRN, details))
		}
	}
	return allErrs
}

func validateClusterOSImage(client API, imageName string, region string, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	customImage, _ := client.GetCustomImageByName(context.TODO(), imageName, region)
	if customImage == nil {
		allErrs = append(allErrs, field.NotFound(path.Child("clusterOSImage"), imageName))
	}
	return allErrs
}

func validateNetworking(client API, ic *types.InstallConfig, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	platform := ic.Platform.IBMCloud

	_, err := client.GetVPC(context.TODO(), platform.VPC)
	if err != nil {
		if err.Error() == fmt.Sprintf("vpc not found: \"%s\"", platform.VPC) {
			allErrs = append(allErrs, field.NotFound(path.Child("vpc"), platform.VPC))
		} else {
			allErrs = append(allErrs, field.InternalError(path.Child("vpc"), err))
		}
	}

	allErrs = append(allErrs, validateSubnets(client, ic, platform.Subnets, path)...)

	return allErrs
}

func validateSubnets(client API, ic *types.InstallConfig, subnets []string, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	validZones, err := client.GetVPCZonesForRegion(context.TODO(), ic.Platform.IBMCloud.Region)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(path.Child("subnets"), err))
	}
	sort.Strings(validZones)
	for _, subnet := range subnets {
		allErrs = append(allErrs, validateSubnetZone(client, subnet, validZones, path)...)
	}

	// TODO: IBM: additional subnet validation
	return allErrs
}

func validateSubnetZone(client API, subnetID string, validZones []string, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if subnet, err := client.GetSubnet(context.TODO(), subnetID); err == nil {
		zoneName := *subnet.Zone.Name
		if !contains(validZones, zoneName) {
			allErrs = append(allErrs, field.Invalid(path.Child("subnets"), subnetID, fmt.Sprintf("subnet is not in expected zones: %s", validZones)))
		}
	} else {
		msg := err.Error()
		if msg == "not found" {
			allErrs = append(allErrs, field.NotFound(path.Child("subnets"), subnetID))
		} else {
			allErrs = append(allErrs, field.InternalError(path.Child("subnets"), err))
		}
	}
	return allErrs
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
