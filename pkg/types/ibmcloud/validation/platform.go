package validation

import (
	"os"

	"sort"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/installer/pkg/types/gcp"

	"github.com/openshift/installer/pkg/validate"
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
	validRegionValues = func() []string {
		validValues := make([]string, len(Regions))
		i := 0
		for r := range Regions {
			validValues[i] = r
			i++
		}
		sort.Strings(validValues)
		return validValues
	}()
)

// ValidatePlatform checks that the specified platform is valid.
func ValidatePlatform(p *gcp.Platform, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, ok := Regions[p.Region]; !ok {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("region"), p.Region, validRegionValues))
	}
	if p.DefaultMachinePlatform != nil {
		allErrs = append(allErrs, ValidateMachinePool(p, p.DefaultMachinePlatform, fldPath.Child("defaultMachinePlatform"))...)
		allErrs = append(allErrs, ValidateDefaultDiskType(p.DefaultMachinePlatform, fldPath.Child("defaultMachinePlatform"))...)
	}
	if p.Network != "" {
		if p.ComputeSubnet == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("computeSubnet"), "must provide a compute subnet when a network is specified"))
		}
		if p.ControlPlaneSubnet == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("controlPlaneSubnet"), "must provide a control plane subnet when a network is specified"))
		}
	}
	if (p.ComputeSubnet != "" || p.ControlPlaneSubnet != "") && p.Network == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("network"), "must provide a VPC network when supplying subnets"))
	}

	if oi, ok := os.LookupEnv("OPENSHIFT_INSTALL_OS_IMAGE_OVERRIDE"); ok && oi != "" && len(p.Licenses) > 0 {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("licenses"), "the use of custom image licenses is forbidden if an OPENSHIFT_INSTALL_OS_IMAGE_OVERRIDE is specified"))
	}

	for i, license := range p.Licenses {
		if validate.URIWithProtocol(license, "https") != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("licenses").Index(i), license, "licenses must be URLs (https) only"))
		}
	}

	return allErrs
}
