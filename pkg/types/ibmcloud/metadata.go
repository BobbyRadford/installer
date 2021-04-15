package ibmcloud

// Metadata contains GCP metadata (e.g. for uninstalling the cluster).
type Metadata struct {
	Region          string `json:"region"`
	ResourceGroupID string `json:"resourceGroupID"`
}
