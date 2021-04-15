package ibmcloud

// Platform stores all the global configuration that all machinesets
// use.
type Platform struct {
	// ResourceGroupID is the the resource group that will be used for the cluster.
	ResourceGroupID string `json:"resourceGroupID"`

	// Region specifies the IBM Cloud region where the cluster will be created.
	Region string `json:"region"`
}
