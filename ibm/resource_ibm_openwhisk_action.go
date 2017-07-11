package ibm

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/apache/incubator-openwhisk-client-go/whisk"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceIBMOpenWhiskAction() *schema.Resource {
	return &schema.Resource{
		Create:   resourceIBMOpenWhiskActionCreate,
		Read:     resourceIBMOpenWhiskActionRead,
		Update:   resourceIBMOpenWhiskActionUpdate,
		Delete:   resourceIBMOpenWhiskActionDelete,
		Exists:   resourceIBMOpenWhiskActionExists,
		Importer: &schema.ResourceImporter{},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the action",
			},
			"overwrite": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Overwrite item if it exists. Default is false.",
			},
			"limits": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"timeout": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Timeout in milliseconds",
						},
						"memory": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Memory.",
						},
					},
				},
			},
			"exec": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Container image name when kind is 'blackbox'.",
						},
						"init": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Optional zipfile reference when code kind is 'nodejs'.",
						},
						"code": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Javascript or Swift code to execute when kind is 'nodejs' or 'swift'.",
						},
						"kind": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of action. Possible values: nodejs, blackbox, swift.",
						},
					},
				},
			},
			"publish": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to publish the item or not.",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "0.0.1",
				Description: "Semantic version of the item.",
			},
			"annotations": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Annotations on the item.",
				ValidateFunc:     validateJSONString,
				DiffSuppressFunc: suppressEquivalentJSON,
				StateFunc: func(v interface{}) string {
					json, _ := normalizeJSONString(v)
					return json
				},
			},
			"parameters": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "[]",
				Description:      "Parameter bindings included in the context passed to the action.",
				ValidateFunc:     validateJSONString,
				DiffSuppressFunc: suppressEquivalentJSON,
				StateFunc: func(v interface{}) string {
					json, _ := normalizeJSONString(v)
					return json
				},
			},
		},
	}
}

func resourceIBMOpenWhiskActionCreate(d *schema.ResourceData, meta interface{}) error {
	wskClient, err := meta.(ClientSession).OpenWhiskClient()
	if err != nil {
		return err
	}
	actionService := wskClient.Actions

	limits := d.Get("limits").([]interface{})
	exec := d.Get("exec").([]interface{})

	payload := whisk.Action{
		Name:      d.Get("name").(string),
		Namespace: wskClient.Namespace,
	}

	if v, ok := d.GetOk("annotations"); ok {
		var err error
		payload.Annotations, err = expandAnnotations(v.(string))
		if err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("parameters"); ok {
		var err error
		payload.Parameters, err = expandParameters(v.(string))
		if err != nil {
			return err
		}
	}

	if publish, ok := d.GetOk("publish"); ok {
		p := publish.(bool)
		payload.Publish = &p
	}

	if version, ok := d.GetOk("version"); ok {
		payload.Version = version.(string)
	}

	payload.Limits = expandLimits(limits)
	payload.Exec = expandExec(exec)

	var overwrite = false
	if ow, ok := d.GetOk("overwrite"); ok {
		overwrite = ow.(bool)
	}

	log.Println("[INFO] Creating OpenWhisk Action")
	action, _, err := actionService.Insert(&payload, overwrite)
	if err != nil {
		return fmt.Errorf("Error creating OpenWhisk Action: %s", err)
	}

	d.SetId(action.Name)
	return resourceIBMOpenWhiskActionRead(d, meta)
}

func resourceIBMOpenWhiskActionRead(d *schema.ResourceData, meta interface{}) error {
	wskClient, err := meta.(ClientSession).OpenWhiskClient()
	if err != nil {
		return err
	}

	actionService := wskClient.Actions
	id := d.Id()

	action, _, err := actionService.Get(id)
	if err != nil {
		return fmt.Errorf("Error retrieving OpenWhisk Action %s : %s", id, err)
	}

	d.SetId(action.Name)
	d.Set("name", action.Name)
	d.Set("limits", flattenLimits(action.Limits))
	d.Set("exec", flattenExec(action.Exec))
	d.Set("publish", action.Publish)
	d.Set("version", action.Version)
	annotations, err := flattenAnnotations(action.Annotations)
	if err != nil {
		return err
	}
	d.Set("annotations", annotations)
	parameters, err := flattenParameters(action.Parameters)
	if err != nil {
		return err
	}
	d.Set("parameters", parameters)
	return nil
}

func resourceIBMOpenWhiskActionUpdate(d *schema.ResourceData, meta interface{}) error {
	wskClient, err := meta.(ClientSession).OpenWhiskClient()
	if err != nil {
		return err
	}
	actionService := wskClient.Actions
	payload := whisk.Action{}

	if d.HasChange("publish") {
		p := d.Get("publish").(bool)
		payload.Publish = &p
	}

	if d.HasChange("version") {
		payload.Version = d.Get("version").(string)
	}
	var overwrite = false
	if ow, ok := d.GetOk("overwrite"); ok {
		overwrite = ow.(bool)
	}

	log.Println("[INFO] Update OpenWhisk Action")

	_, _, err = actionService.Insert(&payload, overwrite)
	if err != nil {
		return fmt.Errorf("Error updating OpenWhisk Action: %s", err)
	}

	return resourceIBMOpenWhiskActionRead(d, meta)
}

func resourceIBMOpenWhiskActionDelete(d *schema.ResourceData, meta interface{}) error {
	wskClient, err := meta.(ClientSession).OpenWhiskClient()
	if err != nil {
		return err
	}
	actionService := wskClient.Actions
	id := d.Id()
	_, err = actionService.Delete(id)
	if err != nil {
		return fmt.Errorf("Error deleting OpenWhisk Action: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceIBMOpenWhiskActionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	wskClient, err := meta.(ClientSession).OpenWhiskClient()
	if err != nil {
		return false, err
	}
	actionService := wskClient.Actions
	id := d.Id()
	action, resp, err := actionService.Get(id)
	if err != nil {
		if resp.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("Error communicating with OpenWhisk Client : %s", err)
	}
	return action.Name == id, nil
}

func expandLimits(l []interface{}) *whisk.Limits {
	if len(l) == 0 || l[0] == nil {
		return &whisk.Limits{}
	}
	in := l[0].(map[string]interface{})
	obj := &whisk.Limits{
		Timeout: ptrToInt(in["timeout"].(int)),
		Memory:  ptrToInt(in["memory"].(int)),
	}
	return obj
}

func flattenLimits(in *whisk.Limits) []interface{} {
	att := make(map[string]interface{})
	if in.Timeout != nil {
		att["timeout"] = in.Timeout
	}
	if in.Memory != nil {
		att["memory"] = in.Memory
	}
	return []interface{}{att}
}

func expandExec(l []interface{}) *whisk.Exec {
	if len(l) == 0 || l[0] == nil {
		return &whisk.Exec{}
	}
	in := l[0].(map[string]interface{})
	obj := &whisk.Exec{
		Image: in["image"].(string),
		Init:  in["init"].(string),
		Code:  ptrToString(in["code"].(string)),
		Kind:  in["kind"].(string),
	}
	return obj
}

func flattenExec(in *whisk.Exec) []interface{} {
	att := make(map[string]interface{})
	if in.Image != "" {
		att["image"] = in.Image
	}
	if in.Init != "" {
		att["init"] = in.Init
	}
	if in.Code != nil {
		att["code"] = *in.Code
	}
	if in.Kind != "" {
		att["kind"] = in.Kind
	}

	return []interface{}{att}
}

func expandAnnotations(annotations string) (whisk.KeyValueArr, error) {
	var result whisk.KeyValueArr
	dc := json.NewDecoder(strings.NewReader(annotations))
	dc.UseNumber()
	err := dc.Decode(&result)
	return result, err
}

func flattenAnnotations(in whisk.KeyValueArr) (string, error) {
	noExec := make(whisk.KeyValueArr, 0, len(in))
	for _, v := range in {
		if v.Key == "exec" {
			continue
		}
		noExec = append(noExec, v)
	}
	b, err := json.Marshal(noExec)
	if err != nil {
		return "", err
	}
	return string(b[:]), nil
}

func expandParameters(annotations string) (whisk.KeyValueArr, error) {
	var result whisk.KeyValueArr
	dc := json.NewDecoder(strings.NewReader(annotations))
	dc.UseNumber()
	err := dc.Decode(&result)
	return result, err
}

func flattenParameters(in whisk.KeyValueArr) (string, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(b[:]), nil
}

func ptrToInt(i int) *int {
	return &i
}

func ptrToString(s string) *string {
	return &s
}
