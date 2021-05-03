package ibmcloud

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM-Cloud/bluemix-go/api/cis/cisv1"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/controller"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/management"
	"github.com/IBM-Cloud/bluemix-go/bmxerror"
	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM-Cloud/bluemix-go/session"
	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	ibmcloudtypes "github.com/openshift/installer/pkg/types/ibmcloud"
	"github.com/pkg/errors"
)

//go:generate mockgen -source=./client.go -destination=./mock/ibmcloudclient_generated.go -package=mock

// API represents the calls made to the API.
type API interface {
	GetCISInstance(ctx context.Context, crnstr string) (*models.ServiceInstance, error)
	GetCustomImageByName(ctx context.Context, imageName string) (*vpcv1.Image, error)
	GetCustomImages(ctx context.Context) ([]vpcv1.Image, error)
	GetDNSZones(ctx context.Context) ([]ibmcloudtypes.DNSZoneResponse, error)
	GetResourceGroups(ctx context.Context) ([]models.ResourceGroup, error)
	GetResourceGroup(ctx context.Context, nameOrID string) (*models.ResourceGroup, error)
	GetSubnet(ctx context.Context, subnetID string) (*vpcv1.Subnet, error)
	GetVPCByName(ctx context.Context, vpcName string, resourceGroupID string) (*vpcv1.VPC, error)
	GetVPCZonesForRegion(ctx context.Context, region string) ([]string, error)
	GetZoneIDByName(ctx context.Context, crn string, name string) (string, error)
}

// Client makes calls to the IBM Cloud API.
type Client struct {
	ssn           *session.Session
	managementAPI management.ResourceManagementAPI
	controllerAPI controller.ResourceControllerAPI
	cisAPI        cisv1.CisServiceAPI
	vpcAPI        *vpcv1.VpcV1
}

// cisServiceID is the Cloud Internet Services' catalog service ID.
var cisServiceID = "75874a60-cb12-11e7-948e-37ac098eb1b9"

// NewClient initializes a client with a session.
func NewClient(ctx context.Context) (*Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	var err error
	ssn, err := GetSession(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	client := &Client{
		ssn: ssn,
	}

	err = client.loadCloudAPIs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load ibm cloud apis")
	}

	return client, nil
}

func (c *Client) loadCloudAPIs() error {
	var apisToLoad []func() error
	apisToLoad = append(apisToLoad, c.loadResourceManagementAPI)
	apisToLoad = append(apisToLoad, c.loadResourceControllerAPI)
	apisToLoad = append(apisToLoad, c.loadCloudInternetServicesAPI)
	apisToLoad = append(apisToLoad, c.loadVPCV1API)

	// Call all the load functions.
	var err error
	for _, fn := range apisToLoad {
		err = fn()
		if err != nil {
			break
		}
	}
	if err != nil {
		return err
	}

	return nil
}

// GetCISInstance gets a specific Cloud Internet Services instance by its CRN.
func (c *Client) GetCISInstance(ctx context.Context, crnstr string) (*models.ServiceInstance, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resourceController := c.controllerAPI.ResourceServiceInstance()
	cisInstance, err := resourceController.GetInstance(crnstr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cis instances")
	}
	return &cisInstance, nil
}

// GetDNSZones returns all of the DNS zones managed by CIS.
func (c *Client) GetDNSZones(ctx context.Context) ([]ibmcloudtypes.DNSZoneResponse, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resourceController := c.controllerAPI.ResourceServiceInstance()

	cisInstancesQuery := controller.ServiceInstanceQuery{
		ServiceID: cisServiceID,
	}

	cisInstances, err := resourceController.ListInstances(cisInstancesQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cis instance")
	}

	var allZones []ibmcloudtypes.DNSZoneResponse
	zonesAPI := c.cisAPI.Zones()
	for _, instance := range cisInstances {
		crnstr := instance.Crn.String()
		zones, err := zonesAPI.ListZones(crnstr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list dns zones")
		}

		for _, zone := range zones {
			zoneStruct := ibmcloudtypes.DNSZoneResponse{
				Name:            zone.Name,
				CISInstanceCRN:  instance.Crn.String(),
				CISInstanceName: instance.Name,
			}
			allZones = append(allZones, zoneStruct)
		}
	}

	return allZones, nil
}

// GetZoneIDByName gets the CIS zone ID from its domain name.
func (c *Client) GetZoneIDByName(ctx context.Context, crn string, name string) (string, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	zones, err := c.cisAPI.Zones().ListZones(crn)
	if err != nil {
		return "", err
	}
	if len(zones) == 0 {
		return "", fmt.Errorf("zone not found: %s", name)
	}

	var zoneID string
	for _, z := range zones {
		if z.Name == name && z.Status == "active" {
			zoneID = z.Id
			break
		}
	}

	if zoneID == "" {
		return "", fmt.Errorf("zone not found: %s", name)
	}
	return zoneID, nil
}

// GetResourceGroup gets a resource group by its name or ID.
func (c *Client) GetResourceGroup(ctx context.Context, nameOrID string) (*models.ResourceGroup, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resourceGroupAPI := c.managementAPI.ResourceGroup()
	query := &management.ResourceGroupQuery{}
	// FindByName() returns a slice of groups, but in reality you cannot have
	// multiple resource groups in a single account with duplicate names. As a
	// result we assume the slice will always have one element.
	groups, err := resourceGroupAPI.FindByName(query, nameOrID)
	if err != nil {
		if bmxe, ok := err.(bmxerror.Error); ok {
			return nil, fmt.Errorf(bmxe.Description())
		}
		return nil, err
	}
	return &groups[0], nil
}

// GetResourceGroups gets the list of resource groups.
func (c *Client) GetResourceGroups(ctx context.Context) ([]models.ResourceGroup, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resourceGroupAPI := c.managementAPI.ResourceGroup()
	query := &management.ResourceGroupQuery{}
	groups, err := resourceGroupAPI.List(query)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// GetSubnet gets a subnet by its ID.
func (c *Client) GetSubnet(ctx context.Context, subnetID string) (*vpcv1.Subnet, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	subnet, _, err := c.vpcAPI.GetSubnet(&vpcv1.GetSubnetOptions{ID: &subnetID})
	if err != nil {
		return nil, err
	}
	return subnet, nil
}

// GetCustomImages gets a list of custom images across all regions. If the image
// status is not "available" it is omitted.
func (c *Client) GetCustomImages(ctx context.Context) ([]vpcv1.Image, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	regions, err := c.getVPCRegions(ctx)
	if err != nil {
		return nil, err
	}

	images := []vpcv1.Image{}
	for _, region := range regions {
		privateImages, err := c.listPrivateImagesForRegion(ctx, region)
		if err != nil {
			return nil, err
		}
		for _, image := range privateImages {
			if *image.Status == vpcv1.ImageStatusAvailableConst {
				images = append(images, image)
			}
		}
	}
	return images, nil
}

// GetCustomImageByName gets a custom image using its name. All regions will be
// searched.
func (c *Client) GetCustomImageByName(ctx context.Context, imageName string) (*vpcv1.Image, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	customImages, err := c.GetCustomImages(ctx)
	if err != nil {
		return nil, err
	}

	var image *vpcv1.Image
	for _, i := range customImages {
		if *i.Name == imageName {
			image = &i
			break
		}
	}

	if image == nil {
		return nil, fmt.Errorf("image not found: %s", imageName)
	}
	return image, nil
}

// GetVPCByName gets a VPC by name within a specific resource group.
func (c *Client) GetVPCByName(ctx context.Context, vpcName string, resourceGroupID string) (*vpcv1.VPC, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	regions, err := c.getVPCRegions(ctx)
	if err != nil {
		return nil, err
	}

	vpcOptions := &vpcv1.ListVpcsOptions{ResourceGroupID: &resourceGroupID}

	for _, region := range regions {
		var foundVPC *vpcv1.VPC
		err := c.vpcAPI.SetServiceURL(fmt.Sprintf("%s/v1", *region.Endpoint))
		if err != nil {
			return nil, errors.Wrap(err, "failed to set vpc api service url")
		}

		vpcCollection, _, err := c.vpcAPI.ListVpcsWithContext(ctx, vpcOptions)
		if err != nil {
			return nil, err
		}

		for _, vpc := range vpcCollection.Vpcs {
			if *vpc.Name == vpcName {
				foundVPC = &vpc
				break
			}
		}

		if foundVPC != nil {
			return foundVPC, nil
		}
	}

	return nil, fmt.Errorf("vpc not found: %s", vpcName)
}

// GetVPCZonesForRegion gets the supported zones for a VPC region.
func (c *Client) GetVPCZonesForRegion(ctx context.Context, region string) ([]string, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	regionZonesOptions := c.vpcAPI.NewListRegionZonesOptions(region)
	zones, _, err := c.vpcAPI.ListRegionZonesWithContext(ctx, regionZonesOptions)
	if err != nil {
		return nil, err
	}

	response := make([]string, len(zones.Zones))
	for idx, zone := range zones.Zones {
		response[idx] = *zone.Name
	}
	return response, err
}

func (c *Client) listPrivateImagesForRegion(ctx context.Context, region vpcv1.Region) ([]vpcv1.Image, error) {
	listImageOptions := c.vpcAPI.NewListImagesOptions()
	listImageOptions.SetVisibility(vpcv1.ImageVisibilityPrivateConst)

	err := c.vpcAPI.SetServiceURL(fmt.Sprintf("%s/v1", *region.Endpoint))
	if err != nil {
		return nil, errors.Wrap(err, "failed to set vpc api service url")
	}

	listImagesResponse, _, err := c.vpcAPI.ListImagesWithContext(ctx, listImageOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list vpc images")
	}

	return listImagesResponse.Images, nil
}

func (c *Client) getVPCRegions(ctx context.Context) ([]vpcv1.Region, error) {
	listRegionsOptions := c.vpcAPI.NewListRegionsOptions()
	listRegionsResponse, _, err := c.vpcAPI.ListRegionsWithContext(ctx, listRegionsOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list vpc regions")
	}

	return listRegionsResponse.Regions, nil
}

func (c *Client) loadResourceManagementAPI() error {
	api, err := management.New(c.ssn)
	if err != nil {
		return errors.Wrap(err, "failed to load resource management apis")
	}
	c.managementAPI = api
	return nil
}

func (c *Client) loadResourceControllerAPI() error {
	api, err := controller.New(c.ssn)
	if err != nil {
		return errors.Wrap(err, "failed to load resource controller apis")
	}
	c.controllerAPI = api
	return nil
}

func (c *Client) loadCloudInternetServicesAPI() error {
	api, err := cisv1.New(c.ssn)
	if err != nil {
		return errors.Wrap(err, "failed to load internet services apis")
	}
	c.cisAPI = api
	return nil
}

func (c *Client) loadVPCV1API() error {
	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: &core.IamAuthenticator{
			ApiKey: c.ssn.Config.BluemixAPIKey,
		},
	})
	if err != nil {
		return err
	}
	c.vpcAPI = vpcService
	return nil
}
