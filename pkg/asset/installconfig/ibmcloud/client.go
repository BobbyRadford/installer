package ibmcloud

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM-Cloud/bluemix-go/api/cis/cisv1"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/controller"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/management"
	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM-Cloud/bluemix-go/session"
	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/pkg/errors"
)

//go:generate mockgen -source=./client.go -destination=./mock/ibmcloudclient_generated.go -package=mock

// API represents the calls made to the API.
type API interface {
	GetCustomImages(ctx context.Context) ([]vpcv1.Image, error)
	GetDNSZones(ctx context.Context) ([]DNSZonesResponse, error)
	GetResourceGroups(ctx context.Context) ([]models.ResourceGroup, error)
}

// Client makes calls to the GCP API.
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

// DNSZonesResponse is the response type for the GetPublicZones function.
type DNSZonesResponse struct {
	Name            string
	CISInstanceCRN  string
	CISInstanceName string
}

// GetDNSZones returns all of the DNS zones managed by CIS.
func (c *Client) GetDNSZones(ctx context.Context) ([]DNSZonesResponse, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resourceController := c.controllerAPI.ResourceServiceInstance()

	cisInstancesQuery := controller.ServiceInstanceQuery{
		ServiceID: cisServiceID,
	}

	cisInstances, err := resourceController.ListInstances(cisInstancesQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cis instances")
	}

	var allZones []DNSZonesResponse
	zonesAPI := c.cisAPI.Zones()
	for _, instance := range cisInstances {
		crnstr := instance.Crn.String()
		zones, err := zonesAPI.ListZones(crnstr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list dns zones")
		}

		for _, zone := range zones {
			zoneStruct := DNSZonesResponse{
				Name:            zone.Name,
				CISInstanceCRN:  instance.Crn.String(),
				CISInstanceName: instance.Name,
			}
			allZones = append(allZones, zoneStruct)
		}
	}

	return allZones, nil
}

// GetResourceGroups gets the list of resource groups.
func (c *Client) GetResourceGroups(ctx context.Context) ([]models.ResourceGroup, error) {
	resourceGroupAPI := c.managementAPI.ResourceGroup()
	query := &management.ResourceGroupQuery{}
	groups, err := resourceGroupAPI.List(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list resource groups")
	}
	return groups, nil
}

// GetCustomImages gets a list of custom images across all regions. If the image
// status is not "available" it is omitted.
func (c *Client) GetCustomImages(ctx context.Context) ([]vpcv1.Image, error) {
	regions, err := c.getVPCRegions(ctx)
	if err != nil {
		return nil, err
	}

	images := []vpcv1.Image{}
	for _, region := range regions {
		privateImages, err := c.listPrivateImagesForRegion(ctx, region)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list vpc images")
		}
		for _, image := range privateImages {
			if *image.Status == vpcv1.ImageStatusAvailableConst {
				images = append(images, image)
			}
		}
	}
	return images, nil
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
