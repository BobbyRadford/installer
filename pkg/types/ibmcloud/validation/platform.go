package validation

import (
	"github.com/IBM-Cloud/bluemix-go/crn"
	"github.com/openshift/installer/pkg/types/ibmcloud"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	// Regions is a map of IBM Cloud regions where VPCs are supported.
	// The key of the map is the short name of the region. The value
	// of the map is the long name of the region.
	Regions = map[string]string{
		// https://cloud.ibm.com/docs/vpc?topic=vpc-creating-a-vpc-in-a-different-region
		"us-south": "US South (Dallas)",
		"us-east":  "US East (Washington DC)",
		"eu-gb":    "United Kindom (London)",
		"eu-de":    "EU Germany (Frankfurt)",
		"jp-tok":   "Japan (Tokyo)",
		"jp-osa":   "Japan (Osaka)",
		"au-syd":   "Australia (Sydney)",
		"ca-tor":   "Canada (Toronto)",
	}

	regionShortNames = func() []string {
		keys := make([]string, len(Regions))
		i := 0
		for r := range Regions {
			keys[i] = r
			i++
		}
		return keys
	}()
)

// ValidatePlatform checks that the specified platform is valid.
func ValidatePlatform(p *ibmcloud.Platform, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if p.Region == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), "region must be specified"))
	} else if _, ok := Regions[p.Region]; !ok {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("region"), p.Region, regionShortNames))
	}

	if p.ClusterOSImage == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("clusterOSImage"), "clusterOSImage must be specified"))
	}

	if p.CISInstanceCRN == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("cisInstanceCRN"), "cisInstanceCRN must be specified"))
	} else {
		_, parseErr := crn.Parse(p.CISInstanceCRN)
		if parseErr != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("cisInstanceCRN"), p.CISInstanceCRN, "cisInstanceCRN is not a valid IBM CRN"))
		}
	}

	allErrs = append(allErrs, ValidateVPCConfig(p, fldPath)...)

	if p.DefaultMachinePlatform != nil {
		allErrs = append(allErrs, ValidateMachinePool(p, p.DefaultMachinePlatform, fldPath.Child("defaultMachinePlatform"))...)
	}
	return allErrs
}

// ValidateMachinePool ...
func ValidateMachinePool(p *ibmcloud.Platform, defaultMachinePlatform *ibmcloud.MachinePool, path *field.Path) field.ErrorList {
	// TODO: IBM: machine pool validation
	return field.ErrorList{}
}

// ValidateVPCConfig ...
func ValidateVPCConfig(p *ibmcloud.Platform, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	details := "if either one of the vpc or subnets fields is defined, they both must be defined"
	if p.VPC != "" || len(p.Subnets) > 0 {
		if p.VPC == "" {
			allErrs = append(allErrs, field.Required(path.Child("vpc"), details))
		}
		if len(p.Subnets) == 0 {
			allErrs = append(allErrs, field.Required(path.Child("subnets"), details))
		}
	}
	return allErrs
}
