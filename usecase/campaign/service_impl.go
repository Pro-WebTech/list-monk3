package campaign

import (
	"encoding/json"
	"fmt"
	"log"
)

func (c *Campaign) GetListMessengers(lo *log.Logger) (resp []MessengersResponse, err error) {
	settings, err := c.sdb.FindByKey(c.db, "providers")
	if err != nil {
		lo.Println("err queryDB[getListMessengers]: ", err)
		return
	}

	out := []MessengersResponse{}
	err = json.Unmarshal([]byte(settings.Value), &out)
	if err != nil {
		lo.Println("err Unmarshal[getListMessengers]: ", err)
		return
	}

	for _, each := range out {
		products := []ListProduct{}
		for _, listP := range each.Product {
			products = append(products, ListProduct{
				Name:  listP.Name,
				Value: fmt.Sprintf("%v_%v", each.Messenger, listP.Name),
			})
		}

		resp = append(resp, MessengersResponse{
			Name:      each.Name,
			Messenger: each.Messenger,
			Product:   products,
		})
	}

	return
}
