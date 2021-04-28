package ibmcloud

import (
	"context"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/installer/pkg/types"
	"github.com/sirupsen/logrus"
)

// Validate executes platform-specific validation.
func Validate(client API, ic *types.InstallConfig) error {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateResourceGroup(client, ic, field.NewPath("platform").Child("ibmcloud"))...)

	return allErrs.ToAggregate()
}

func validateResourceGroup(client API, ic *types.InstallConfig, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if ic.IBMCloud.ResourceGroup != "" {
		resourceGroups, err := client.GetResourceGroups(context.TODO())
		if err != nil {
			return append(allErrs, field.InternalError(fieldPath.Child("resourceGroup"), err))
		}

		found := false
		for _, rg := range resourceGroups {
			logrus.Infof("matching... %s to %s", rg.ID, ic.IBMCloud.ResourceGroup)
			if rg.ID == ic.IBMCloud.ResourceGroup {
				found = true
			}
		}

		if !found {
			return append(allErrs, field.Invalid(fieldPath.Child("resourceGroup"), ic.IBMCloud.ResourceGroup, "invalid resource group ID"))
		}
	}

	return allErrs
}
