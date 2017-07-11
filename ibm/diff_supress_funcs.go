package ibm

import (
	"encoding/json"
	"log"
	"reflect"

	"github.com/hashicorp/terraform/helper/schema"
)

func suppressEquivalentJSON(k, old, new string, d *schema.ResourceData) bool {
	var oldObj, newObj interface{}
	err := json.Unmarshal([]byte(old), &oldObj)
	if err != nil {
		log.Printf("Error mashalling string 1 :: %s", err.Error())
		return false
	}
	err = json.Unmarshal([]byte(new), &newObj)
	if err != nil {
		log.Printf("Error mashalling string 2 :: %s", err.Error())
		return false
	}
	return reflect.DeepEqual(oldObj, newObj)
}
