package ibmcloud

import (
	"context"
	"time"

	"github.com/IBM-Cloud/bluemix-go/api/cis/cisv1"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/controller"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/management"
	"github.com/IBM-Cloud/bluemix-go/models"
	"github.com/IBM-Cloud/bluemix-go/session"
	"github.com/pkg/errors"
)

//go:generate mockgen -source=./client.go -destination=./mock/ibmcloudclient_generated.go -package=mock

// API represents the calls made to the API.
type API interface {
	GetPublicDomains(ctx context.Context, project string) ([]string, error)
	GetResourceGroups(ctx context.Context) ([]models.ResourceGroup, error)
}

// Client makes calls to the GCP API.
type Client struct {
	ssn           *session.Session
	managementApi management.ResourceManagementAPI
	controllerApi controller.ResourceControllerAPI
	cisApi        cisv1.CisServiceAPI
}

var cisServiceId = "75874a60-cb12-11e7-948e-37ac098eb1b9"

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
	apisToLoad = append(apisToLoad, c.loadResourceManagementApi)
	apisToLoad = append(apisToLoad, c.loadResourceControllerApi)
	apisToLoad = append(apisToLoad, c.loadCloudInternetServicesApi)

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

func (c *Client) GetPublicDomains(ctx context.Context, project string) ([]string, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resourceController := c.controllerApi.ResourceServiceInstance()

	cisInstancesQuery := controller.ServiceInstanceQuery{
		ServiceID: cisServiceId,
	}

	cisInstances, err := resourceController.ListInstances(cisInstancesQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cis instances")
	}

	var allZones []cisv1.Zone
	zonesApi := c.cisApi.Zones()
	for _, instance := range cisInstances {
		crnstr := instance.Crn.String()
		zones, err := zonesApi.ListZones(crnstr)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list dns zones")
		}

		allZones = append(allZones, zones...)
	}

	var zoneNames []string
	for _, zone := range allZones {
		zoneNames = append(zoneNames, zone.Name)
	}

	return zoneNames, nil
}

// GetResourceGroups gets the list of resource groups
func (c *Client) GetResourceGroups(ctx context.Context) ([]models.ResourceGroup, error) {
	resourceGroupApi := c.managementApi.ResourceGroup()
	query := &management.ResourceGroupQuery{}
	groups, err := resourceGroupApi.List(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list resource groups")
	}
	return groups, nil
}

func (c *Client) loadResourceManagementApi() error {
	api, err := management.New(c.ssn)
	if err != nil {
		return errors.Wrap(err, "failed to load resource management apis")
	}
	c.managementApi = api
	return nil
}

func (c *Client) loadResourceControllerApi() error {
	api, err := controller.New(c.ssn)
	if err != nil {
		return errors.Wrap(err, "failed to load resource controller apis")
	}
	c.controllerApi = api
	return nil
}

func (c *Client) loadCloudInternetServicesApi() error {
	api, err := cisv1.New(c.ssn)
	if err != nil {
		return errors.Wrap(err, "failed to load internet services apis")
	}
	c.cisApi = api
	return nil
}
