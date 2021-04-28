package ibmcloud

// Platform stores all the global configuration that all machinesets use.
type Platform struct {
	// Region specifies the IBM Cloud region where the cluster will be
	// created.
	Region string `json:"region"`

	// CISInstanceCRN is the Cloud Internet Services CRN of the base domain DNS
	// zone.
	CISInstanceCRN string `json:"cisInstanceCRN"`

	// ClusterOSImage is the name of the custom RHCOS image.
	ClusterOSImage string `json:"clusterOSImage"`

	// ResourceGroup is the name of an existing resource group where the cluster
	// and all required resources will be created.
	// +optional
	ResourceGroup string `json:"resourceGroup,omitempty"`

	// DefaultMachinePlatform is the default configuration used when installing
	// on IBM Cloud for machine pools which do not define their own platform
	// configuration.
	// +optional
	DefaultMachinePlatform *MachinePool `json:"defaultMachinePlatform,omitempty"`

	// VPC is the name of an existing VPC network.
	// +optional
	VPC string `json:"vpc,omitempty"`

	// VPCResourceGroup is he name of the existing VPC's resource group.
	// +optional
	VPCResourceGroup string `json:"vpcResourceGroup,omitempty"`

	// Subnets is a list of existing subnet IDs. Leave unset and the installer
	// will create new subnets in the VPC network on your behalf.
	// +optional
	Subnets []string `json:"subnets,omitempty"`
}

// SetBaseDomain sets the CISInstanceCRN.
func (p *Platform) SetBaseDomain(cisInstanceCRN string) error {
	p.CISInstanceCRN = cisInstanceCRN
	return nil
}
