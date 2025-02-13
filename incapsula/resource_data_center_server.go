package incapsula

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDataCenterServer() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "This resource is deprecated. It will be removed in a future version. Please use resource incapsula_data_centers_configuration instead.",
		Create:             resourceDataCenterServerCreate,
		Read:               resourceDataCenterServerRead,
		Update:             resourceDataCenterServerUpdate,
		Delete:             resourceDataCenterServerDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idSlice := strings.Split(d.Id(), "/")
				if len(idSlice) != 3 || idSlice[0] == "" || idSlice[1] == "" || idSlice[2] == "" {
					return nil, fmt.Errorf("unexpected format of ID (%q), expected site_id/dc_id/server_id", d.Id())
				}

				siteID := idSlice[0]
				dcID := idSlice[1]
				serverID := idSlice[2]

				d.Set("site_id", siteID)
				d.Set("dc_id", dcID)
				d.SetId(serverID)
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			// Required Arguments
			"dc_id": {
				Description: "Numeric identifier of the data center server to operate on.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"site_id": {
				Description: "Numeric identifier of the site to operate on.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},

			// Optional Arguments
			"server_address": {
				Description: "The server's address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"is_standby": {
				Description: "Set the server as Active (P0) or Standby (P1).",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "false",
			},
			"is_enabled": {
				Description: "Enables the data center server.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "true",
			},
		},
	}
}

func resourceDataCenterServerCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	dataCenterServerAddResponse, err := client.AddDataCenterServer(
		d.Get("dc_id").(string),
		d.Get("server_address").(string),
		d.Get("is_standby").(string),
		d.Get("is_enabled").(string),
	)

	if err != nil {
		return err
	}

	// Set the server ID
	d.SetId(dataCenterServerAddResponse.ServerID)

	return resourceDataCenterServerRead(d, m)
}

func resourceDataCenterServerRead(d *schema.ResourceData, m interface{}) error {
	// Implement by reading the ListDataCentersResponse for the data centers
	client := m.(*Client)

	listDataCentersResponse, err := client.ListDataCenters(d.Get("site_id").(string))

	// List data centers response object may indicate that the Site ID has been deleted (9413)
	if listDataCentersResponse != nil {
		// Res can oscillate between strings and ints
		var resString string
		if resNumber, ok := listDataCentersResponse.Res.(float64); ok {
			resString = fmt.Sprintf("%d", int(resNumber))
		} else {
			resString = listDataCentersResponse.Res.(string)
		}
		if resString == "9413" {
			log.Printf("[INFO] Incapsula Site ID %s has already been deleted: %s\n", d.Get("site_id"), err)
			d.SetId("")
			return nil
		}
	}

	if err != nil {
		return err
	}

	found := false

	serverAddressToFind := d.Get("server_address").(string)

	for _, dataCenter := range listDataCentersResponse.DCs {
		if dataCenter.ID == d.Get("dc_id").(string) {
			for _, server := range dataCenter.Servers {
				// Find data center server by ID or by address
				// If data centers were edited manually in the portal
				// Their IDs will change and not match tfstate
				// But server's address must be unique across DCs
				if server.ID == d.Id() || server.Address == serverAddressToFind {
					d.Set("is_enabled", server.Enabled)
					d.Set("server_address", server.Address)
					d.Set("is_standby", server.IsStandBy)
					d.SetId(server.ID)
					found = true
				}
			}
		}
	}

	if !found {
		log.Printf("[INFO] Incapsula Data Center Server ID %s for Site ID %s has already been deleted: %s\n", d.Id(), d.Get("site_id"), err)
		d.SetId("")
		return nil
	}

	return nil
}

func resourceDataCenterServerUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	_, err := client.EditDataCenterServer(
		d.Id(),
		d.Get("server_address").(string),
		d.Get("is_standby").(string),
		d.Get("is_enabled").(string),
	)

	if err != nil {
		return err
	}

	return nil
}

func resourceDataCenterServerDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	serverID := d.Id()
	err := client.DeleteDataCenterServer(serverID)

	if err != nil {
		return err
	}

	// Set the ID to empty
	// Implicitly clears the resource
	d.SetId("")

	return nil
}
