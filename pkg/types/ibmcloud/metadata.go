package ibmcloud

// Metadata contains IBM Cloud metadata (e.g. for uninstalling the cluster).
type Metadata struct {
	Region        string `json:"region"`
	ResourceGroup string `json:"resourceGroup"`
}
